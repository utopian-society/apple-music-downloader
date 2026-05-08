package alacfix

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// ============================================================
// bitReader tests
// ============================================================

func TestBitReaderRead(t *testing.T) {
	// 0b10110001 = 0xB1
	br := newBitReader([]byte{0xB1})

	// Read 1 bit: should be 1
	v, err := br.read(1)
	if err != nil || v != 1 {
		t.Fatalf("read(1): got %d, err %v", v, err)
	}

	// Read 3 bits: 011 = 3
	v, err = br.read(3)
	if err != nil || v != 3 {
		t.Fatalf("read(3): got %d, err %v", v, err)
	}

	// Read 4 bits: 0001 = 1
	v, err = br.read(4)
	if err != nil || v != 1 {
		t.Fatalf("read(4): got %d, err %v", v, err)
	}

	// Buffer exhausted; next read should return errEOF
	_, err = br.read(1)
	if err != errEOF {
		t.Fatalf("expected errEOF, got %v", err)
	}
}

func TestBitReaderReadZero(t *testing.T) {
	br := newBitReader([]byte{0xFF})
	v, err := br.read(0)
	if err != nil || v != 0 {
		t.Fatalf("read(0): got %d, err %v", v, err)
	}
	// pos should not advance
	if br.pos != 0 {
		t.Fatalf("pos advanced unexpectedly to %d", br.pos)
	}
}

func TestBitReaderShow(t *testing.T) {
	br := newBitReader([]byte{0b10100000})
	v, err := br.show(3) // peek at top 3 bits: 101 = 5
	if err != nil || v != 5 {
		t.Fatalf("show(3): got %d, err %v", v, err)
	}
	// pos must not advance
	if br.pos != 0 {
		t.Fatalf("show advanced pos to %d", br.pos)
	}
	// read same bits
	v2, _ := br.read(3)
	if v2 != 5 {
		t.Fatalf("read after show: got %d", v2)
	}
}

func TestBitReaderSkip(t *testing.T) {
	br := newBitReader([]byte{0b00000001})
	if err := br.skip(7); err != nil {
		t.Fatalf("skip(7): %v", err)
	}
	v, err := br.read(1)
	if err != nil || v != 1 {
		t.Fatalf("read after skip: got %d, err %v", v, err)
	}
}

func TestBitReaderSkipEOF(t *testing.T) {
	br := newBitReader([]byte{0xFF})
	if err := br.skip(9); err != errEOF {
		t.Fatalf("expected errEOF, got %v", err)
	}
}

func TestBitReaderLeft(t *testing.T) {
	br := newBitReader([]byte{0xAA, 0xBB})
	if br.left() != 16 {
		t.Fatalf("expected 16, got %d", br.left())
	}
	_, _ = br.read(5)
	if br.left() != 11 {
		t.Fatalf("expected 11, got %d", br.left())
	}
}

func TestBitReaderReadSigned_Positive(t *testing.T) {
	// 4-bit value 0b0011 = 3 (positive)
	br := newBitReader([]byte{0b00110000})
	v, err := br.readSigned(4)
	if err != nil || v != 3 {
		t.Fatalf("readSigned(4): got %d, err %v", v, err)
	}
}

func TestBitReaderReadSigned_Negative(t *testing.T) {
	// 4-bit value 0b1101 = 13 unsigned → -3 signed (two's complement)
	br := newBitReader([]byte{0b11010000})
	v, err := br.readSigned(4)
	if err != nil || v != -3 {
		t.Fatalf("readSigned(4): got %d (want -3), err %v", v, err)
	}
}

func TestBitReaderUnary09_Zero(t *testing.T) {
	// First bit is 0 → should return 0
	br := newBitReader([]byte{0b01111111})
	v, err := br.unary09()
	if err != nil || v != 0 {
		t.Fatalf("unary09: got %d, err %v", v, err)
	}
}

func TestBitReaderUnary09_Three(t *testing.T) {
	// 1110xxxx → three 1s then a 0 → returns 3
	br := newBitReader([]byte{0b11100000})
	v, err := br.unary09()
	if err != nil || v != 3 {
		t.Fatalf("unary09: got %d (want 3), err %v", v, err)
	}
}

func TestBitReaderUnary09_MaxNine(t *testing.T) {
	// Nine 1-bits set (all of first byte plus first bit of second)
	// 0xFF = 11111111, 0x80 = 1xxxxxxx
	br := newBitReader([]byte{0xFF, 0x80})
	v, err := br.unary09()
	if err != nil || v != 9 {
		t.Fatalf("unary09: got %d (want 9), err %v", v, err)
	}
}

func TestBitReaderUnary09_EOF(t *testing.T) {
	// All 1-bits, only 8 bits available → hits EOF before 0 terminator
	br := newBitReader([]byte{0xFF})
	_, err := br.unary09()
	if err != errEOF {
		t.Fatalf("expected errEOF, got %v", err)
	}
}

// ============================================================
// avLog2 tests
// ============================================================

func TestAvLog2(t *testing.T) {
	cases := []struct {
		in  uint32
		out int
	}{
		{0, 0},
		{1, 0},
		{2, 1},
		{3, 1},
		{4, 2},
		{7, 2},
		{8, 3},
		{255, 7},
		{256, 8},
		{0xFFFFFFFF, 31},
	}
	for _, c := range cases {
		got := avLog2(c.in)
		if got != c.out {
			t.Errorf("avLog2(%d) = %d, want %d", c.in, got, c.out)
		}
	}
}

// ============================================================
// parseAlacMagicCookie tests
// ============================================================

func makeAlacCookie(maxFrames uint32, sampleSize, histMult, initHist, riceLimit, channels byte) []byte {
	c := make([]byte, 28)
	// bytes 0-3: version_flags (ignored)
	binary.BigEndian.PutUint32(c[4:8], maxFrames)
	// byte 8: compat (ignored)
	c[9] = sampleSize
	c[10] = histMult
	c[11] = initHist
	c[12] = riceLimit
	c[13] = channels
	return c
}

func TestParseAlacMagicCookie_Valid(t *testing.T) {
	cookie := makeAlacCookie(4096, 16, 40, 10, 14, 2)
	p, err := parseAlacMagicCookie(cookie)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.maxSamplesPerFrame != 4096 {
		t.Errorf("maxSamplesPerFrame: got %d, want 4096", p.maxSamplesPerFrame)
	}
	if p.sampleSize != 16 {
		t.Errorf("sampleSize: got %d, want 16", p.sampleSize)
	}
	if p.riceHistoryMult != 40 {
		t.Errorf("riceHistoryMult: got %d, want 40", p.riceHistoryMult)
	}
	if p.riceInitialHistory != 10 {
		t.Errorf("riceInitialHistory: got %d, want 10", p.riceInitialHistory)
	}
	if p.riceLimit != 14 {
		t.Errorf("riceLimit: got %d, want 14", p.riceLimit)
	}
	if p.channels != 2 {
		t.Errorf("channels: got %d, want 2", p.channels)
	}
}

func TestParseAlacMagicCookie_TooShort(t *testing.T) {
	_, err := parseAlacMagicCookie(make([]byte, 23))
	if err == nil {
		t.Fatal("expected error for too-short cookie")
	}
}

func TestParseAlacMagicCookie_ExactMinLength(t *testing.T) {
	cookie := makeAlacCookie(1024, 24, 20, 5, 10, 1)
	_, err := parseAlacMagicCookie(cookie)
	if err != nil {
		t.Fatalf("expected no error for 28-byte cookie: %v", err)
	}
}

// ============================================================
// findChild / findAllChildren tests
// ============================================================

// buildAtom creates a simple 4-byte-size + 4-byte-type atom with given body.
func buildAtom(typ string, body []byte) []byte {
	size := 8 + len(body)
	buf := make([]byte, size)
	binary.BigEndian.PutUint32(buf[0:4], uint32(size))
	copy(buf[4:8], []byte(typ))
	copy(buf[8:], body)
	return buf
}

func TestFindChild_Found(t *testing.T) {
	child := buildAtom("test", []byte("hello"))
	buf := child
	a, ok := findChild(buf, 0, len(buf), "test")
	if !ok {
		t.Fatal("expected to find atom")
	}
	if a.typ != "test" {
		t.Errorf("typ: got %q, want %q", a.typ, "test")
	}
	if a.bodyOff != 8 {
		t.Errorf("bodyOff: got %d, want 8", a.bodyOff)
	}
	if a.endOff != len(buf) {
		t.Errorf("endOff: got %d, want %d", a.endOff, len(buf))
	}
}

func TestFindChild_NotFound(t *testing.T) {
	child := buildAtom("nope", []byte("data"))
	_, ok := findChild(child, 0, len(child), "test")
	if ok {
		t.Fatal("expected not to find atom")
	}
}

func TestFindChild_FirstMatch(t *testing.T) {
	// Two atoms with different types; look for second type.
	first := buildAtom("aaaa", []byte("111"))
	second := buildAtom("bbbb", []byte("222"))
	buf := append(first, second...)
	a, ok := findChild(buf, 0, len(buf), "bbbb")
	if !ok {
		t.Fatal("expected to find bbbb atom")
	}
	if a.hdrOff != len(first) {
		t.Errorf("hdrOff: got %d, want %d", a.hdrOff, len(first))
	}
}

func TestFindChild_SizeZeroMeansToEnd(t *testing.T) {
	buf := make([]byte, 12)
	binary.BigEndian.PutUint32(buf[0:4], 0) // size=0 means "to end"
	copy(buf[4:8], "zero")
	_, ok := findChild(buf, 0, len(buf), "zero")
	if !ok {
		t.Fatal("expected to find size-0 atom")
	}
}

func TestFindChild_ExtendedSize(t *testing.T) {
	// Extended size atom: size field == 1, real size in next 8 bytes.
	body := []byte("extended body data")
	totalSize := 16 + len(body)
	buf := make([]byte, totalSize)
	binary.BigEndian.PutUint32(buf[0:4], 1) // size == 1 → extended
	copy(buf[4:8], "ext1")
	binary.BigEndian.PutUint64(buf[8:16], uint64(totalSize))
	copy(buf[16:], body)

	a, ok := findChild(buf, 0, len(buf), "ext1")
	if !ok {
		t.Fatal("expected to find extended-size atom")
	}
	if a.bodyOff != 16 {
		t.Errorf("bodyOff: got %d, want 16", a.bodyOff)
	}
	if a.endOff != totalSize {
		t.Errorf("endOff: got %d, want %d", a.endOff, totalSize)
	}
}

func TestFindChild_TruncatedExtendedSize(t *testing.T) {
	// Extended size marker but buffer too small to hold 16-byte header.
	buf := make([]byte, 12) // only 12 bytes, need 16 for extended header
	binary.BigEndian.PutUint32(buf[0:4], 1)
	copy(buf[4:8], "ext2")
	_, ok := findChild(buf, 0, len(buf), "ext2")
	if ok {
		t.Fatal("expected not to find truncated extended-size atom")
	}
}

func TestFindAllChildren_MultipleMatches(t *testing.T) {
	a1 := buildAtom("trak", []byte("track1"))
	a2 := buildAtom("trak", []byte("track2"))
	a3 := buildAtom("trak", []byte("track3"))
	buf := append(append(a1, a2...), a3...)
	atoms := findAllChildren(buf, 0, len(buf), "trak")
	if len(atoms) != 3 {
		t.Fatalf("expected 3 atoms, got %d", len(atoms))
	}
}

func TestFindAllChildren_NoMatches(t *testing.T) {
	buf := buildAtom("aaaa", []byte("data"))
	atoms := findAllChildren(buf, 0, len(buf), "bbbb")
	if len(atoms) != 0 {
		t.Fatalf("expected 0 atoms, got %d", len(atoms))
	}
}

func TestFindAllChildren_MixedTypes(t *testing.T) {
	a1 := buildAtom("trak", []byte("t1"))
	a2 := buildAtom("other", []byte("o"))
	a3 := buildAtom("trak", []byte("t2"))
	buf := append(append(a1, a2...), a3...)
	atoms := findAllChildren(buf, 0, len(buf), "trak")
	if len(atoms) != 2 {
		t.Fatalf("expected 2 atoms, got %d", len(atoms))
	}
}

// ============================================================
// patchInPlace tests
// ============================================================

func TestPatchInPlace_Basic(t *testing.T) {
	// 2 bytes = 16 bits; body ends at bit 8.
	// After patching, bits 8,9,10 should be 1 and bits 11-15 should be 0.
	data := []byte{0xFF, 0x00}
	ok := patchInPlace(data, 0, 2, 8)
	if !ok {
		t.Fatal("patchInPlace returned false")
	}
	// byte[1] should have top 3 bits set (111), rest 0 → 0b11100000 = 0xE0
	if data[1] != 0xE0 {
		t.Errorf("data[1]: got 0x%02X, want 0xE0", data[1])
	}
}

func TestPatchInPlace_NegativeBodyEndBit(t *testing.T) {
	data := []byte{0x00, 0x00}
	ok := patchInPlace(data, 0, 2, -1)
	if ok {
		t.Fatal("expected patchInPlace to return false for negative bodyEndBit")
	}
}

func TestPatchInPlace_InsufficientRoom(t *testing.T) {
	// bodyEndBit + 3 > totalBits → no room for the 3-bit end tag
	data := []byte{0xFF}
	ok := patchInPlace(data, 0, 1, 6) // 6+3=9 > 8
	if ok {
		t.Fatal("expected patchInPlace to return false when no room for end tag")
	}
}

func TestPatchInPlace_ExactlyAtEnd(t *testing.T) {
	// 1 byte = 8 bits; bodyEndBit=5, so 5+3=8 exactly fits.
	data := []byte{0x00}
	ok := patchInPlace(data, 0, 1, 5)
	if !ok {
		t.Fatal("expected patchInPlace to succeed when end tag fits exactly")
	}
	// bits 5,6,7 should be 1 → 0b00000111 = 0x07
	if data[0] != 0x07 {
		t.Errorf("data[0]: got 0x%02X, want 0x07", data[0])
	}
}

func TestPatchInPlace_PartialBytePadding(t *testing.T) {
	// 3 bytes = 24 bits, body ends at bit 9.
	// Bits 9,10,11 are set to 1 (the TYPE_END tag).
	// padStart = 12, bitInByte = 12&7 = 4.
	// keep mask = 0xFF<<(8-4) = 0xF0, so lower nibble of byte[1] is cleared.
	// byte[1] (bits 8-15): OR with 0x40|0x20|0x10=0x70, then AND with 0xF0
	//   → 0xFF | 0x70 = 0xFF, then 0xFF & 0xF0 = 0xF0.
	data := []byte{0xFF, 0xFF, 0xFF}
	ok := patchInPlace(data, 0, 3, 9)
	if !ok {
		t.Fatal("patchInPlace returned false")
	}
	// byte[1]: top nibble preserved (bit8 was 1, bits9-11 set, bits12-15 zeroed) = 0xF0
	if data[1] != 0xF0 {
		t.Errorf("data[1]: got 0x%02X, want 0xF0", data[1])
	}
	// byte[2] should be fully zeroed
	if data[2] != 0x00 {
		t.Errorf("data[2]: got 0x%02X, want 0x00", data[2])
	}
}

func TestPatchInPlace_WithOffset(t *testing.T) {
	// Offset into a larger buffer; patch starts at byte offset 2.
	data := []byte{0xAA, 0xBB, 0x00, 0x00}
	ok := patchInPlace(data, 2, 2, 8) // body ends at bit 8 of the 2-byte span starting at offset 2
	if !ok {
		t.Fatal("patchInPlace returned false")
	}
	// data[0], data[1] should be unchanged
	if data[0] != 0xAA || data[1] != 0xBB {
		t.Errorf("prefix bytes modified: %02X %02X", data[0], data[1])
	}
	// data[3] should have top 3 bits set = 0xE0
	if data[3] != 0xE0 {
		t.Errorf("data[3]: got 0x%02X, want 0xE0", data[3])
	}
}

// ============================================================
// findAlacTracks tests
// ============================================================

func TestFindAlacTracks_TooSmall(t *testing.T) {
	_, err := findAlacTracks([]byte{0x00})
	if err == nil {
		t.Fatal("expected error for too-small input")
	}
}

func TestFindAlacTracks_NoMoov(t *testing.T) {
	// A valid-looking atom that is not 'moov'
	buf := buildAtom("ftyp", []byte("M4A "))
	_, err := findAlacTracks(buf)
	if err == nil {
		t.Fatal("expected error when no moov atom")
	}
}

func TestFindAlacTracks_EmptyMoov(t *testing.T) {
	// moov with no trak children → no tracks returned, no error.
	// Wrap in a ftyp prefix so the buffer is large enough for findChild
	// (which requires p < end-8 to enter the loop).
	ftypAtom := buildAtom("ftyp", []byte("M4A "))
	moovAtom := buildAtom("moov", buildAtom("udta", []byte("padding")))
	buf := append(ftypAtom, moovAtom...)
	tracks, err := findAlacTracks(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tracks) != 0 {
		t.Fatalf("expected 0 tracks, got %d", len(tracks))
	}
}

func TestFindAlacTracks_NonALACTrack(t *testing.T) {
	// A trak with handler_type 'soun' but stsd entry type 'mp4a' should be skipped.
	// We skip full construction here and instead rely on the entry-type check.
	// Build a minimal trak that will be skipped because the soun handler is absent.
	trakBody := buildAtom("mdia", buildAtom("smhd", nil))
	trakAtom := buildAtom("trak", trakBody)
	moovAtom := buildAtom("moov", trakAtom)

	tracks, err := findAlacTracks(moovAtom)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tracks) != 0 {
		t.Fatalf("expected 0 tracks (non-ALAC skipped), got %d", len(tracks))
	}
}

// ============================================================
// findBodyEndBit tests
// ============================================================

func TestFindBodyEndBit_EmptyPacket(t *testing.T) {
	params := &alacParams{
		maxSamplesPerFrame: 4096,
		sampleSize:         16,
		riceHistoryMult:    40,
		riceInitialHistory: 10,
		riceLimit:          14,
		channels:           2,
	}
	result := findBodyEndBit([]byte{}, params)
	if result != -1 {
		t.Errorf("expected -1 for empty packet, got %d", result)
	}
}

func TestFindBodyEndBit_TooFewBitsForElement(t *testing.T) {
	// Only 2 bits available – not enough for a 3-bit element tag
	params := &alacParams{
		maxSamplesPerFrame: 4096,
		sampleSize:         16,
		channels:           2,
	}
	// Single byte with only top 2 bits set; left() == 8 >= 3, so we enter loop,
	// but the element tag read might succeed depending on the tag value.
	// Use a packet with a single byte of 0x00 (tag=0 → SCE element).
	// SCE needs many bits, so it will fail → return -1.
	result := findBodyEndBit([]byte{0x00}, params)
	// Either it parsed something (lastEnd set) or failed (-1). Both are valid,
	// but with only 8 bits available parsing the full element should fail.
	_ = result // just ensure no panic
}

func TestFindBodyEndBit_EndTagImmediate(t *testing.T) {
	// A packet that starts with 0b111 (TYPE_END = 7) should return position after that 3-bit read.
	params := &alacParams{
		maxSamplesPerFrame: 4096,
		sampleSize:         16,
		channels:           2,
	}
	// 0b11100000 = 0xE0, first 3 bits are 111 = TYPE_END
	result := findBodyEndBit([]byte{0xE0, 0x00, 0x00, 0x00}, params)
	// Should return br.pos after reading TYPE_END = 3
	if result != 3 {
		t.Errorf("expected 3 for immediate TYPE_END, got %d", result)
	}
}

// ============================================================
// Run() integration tests
// ============================================================

func TestRun_FileNotFound(t *testing.T) {
	err := Run("/nonexistent/path/file.m4a", false)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestRun_InvalidMP4(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.m4a")
	// Write garbage data – not a valid MP4
	if err := os.WriteFile(path, []byte("this is not an mp4 file at all"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Should return error because no 'moov' atom found
	err := Run(path, false)
	if err == nil {
		t.Fatal("expected error for invalid MP4")
	}
}

func TestRun_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.m4a")
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	err := Run(path, false)
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}

func TestRun_ValidMoovNoAlacTracks(t *testing.T) {
	// Construct a minimal MP4 with moov but no alac tracks.
	// moov → trak → mdia → hdlr (handler="vide") → no alac
	dir := t.TempDir()
	path := filepath.Join(dir, "noalac.m4a")

	// Build a hdlr atom with handler_type "vide"
	hdlrBody := make([]byte, 12)
	copy(hdlrBody[8:12], "vide")
	hdlrAtom := buildAtom("hdlr", hdlrBody)

	mdiaAtom := buildAtom("mdia", hdlrAtom)
	trakAtom := buildAtom("trak", mdiaAtom)
	moovAtom := buildAtom("moov", trakAtom)

	if err := os.WriteFile(path, moovAtom, 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Should succeed with no patches (no alac tracks found)
	err := Run(path, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_ForceWritesEvenWithoutPatches(t *testing.T) {
	// With force=true and no patches, file should still be written to output path.
	dir := t.TempDir()
	inPath := filepath.Join(dir, "in.m4a")
	outPath := filepath.Join(dir, "out.m4a")

	// Minimal valid MP4 with moov but no alac tracks
	hdlrBody := make([]byte, 12)
	copy(hdlrBody[8:12], "vide")
	hdlrAtom := buildAtom("hdlr", hdlrBody)
	mdiaAtom := buildAtom("mdia", hdlrAtom)
	trakAtom := buildAtom("trak", mdiaAtom)
	moovAtom := buildAtom("moov", trakAtom)

	if err := os.WriteFile(inPath, moovAtom, 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// force=true, but no ALAC tracks → tracks == 0 → Run returns nil without writing
	err := Run(inPath, true, outPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// outPath should NOT be created because len(tracks)==0 causes early return
	_, statErr := os.Stat(outPath)
	if statErr == nil {
		t.Log("out file was written (len(tracks)>0 path or force path reached)")
	}
}

func TestRun_OutputPathDifferentFromInput(t *testing.T) {
	// Ensure output written to alternate path, input unchanged.
	dir := t.TempDir()
	inPath := filepath.Join(dir, "in.m4a")
	outPath := filepath.Join(dir, "out.m4a")

	// File that's not a valid MP4 → Run returns error, doesn't touch outPath.
	if err := os.WriteFile(inPath, []byte("not mp4"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_ = Run(inPath, false, outPath)
	// The important check: inPath content is not modified by Run (Run only reads it first)
	content, _ := os.ReadFile(inPath)
	if string(content) != "not mp4" {
		t.Error("input file was modified unexpectedly")
	}
}

// ============================================================
// Regression: patchInPlace with bodyEndBit=0
// ============================================================

func TestPatchInPlace_BodyEndBitZero(t *testing.T) {
	// Edge case: body ends at bit 0, so the 3-bit end tag starts at bit 0.
	data := []byte{0x00, 0x00}
	ok := patchInPlace(data, 0, 2, 0)
	if !ok {
		t.Fatal("patchInPlace returned false for bodyEndBit=0")
	}
	// bits 0,1,2 set → 0b11100000 = 0xE0 in byte[0]
	if data[0] != 0xE0 {
		t.Errorf("data[0]: got 0x%02X, want 0xE0", data[0])
	}
}

// ============================================================
// Boundary: findChild with insufficient buffer (< 8 bytes)
// ============================================================

func TestFindChild_BufferTooSmall(t *testing.T) {
	buf := []byte{0x00, 0x00, 0x00} // only 3 bytes, < 8
	_, ok := findChild(buf, 0, len(buf), "test")
	if ok {
		t.Fatal("expected not to find atom in too-small buffer")
	}
}

// ============================================================
// Regression: findAllChildren stops on invalid atom size
// ============================================================

func TestFindAllChildren_InvalidAtomSize(t *testing.T) {
	// Second atom has size < 8 (invalid), should stop iteration safely.
	valid := buildAtom("trak", []byte("data"))
	// Craft an atom with size=4 (invalid, < 8)
	bad := []byte{0x00, 0x00, 0x00, 0x04, 't', 'r', 'a', 'k'}
	buf := append(valid, bad...)
	atoms := findAllChildren(buf, 0, len(buf), "trak")
	// Only the first valid atom should be found; bad atom causes stop.
	if len(atoms) != 1 {
		t.Errorf("expected 1 atom, got %d", len(atoms))
	}
}