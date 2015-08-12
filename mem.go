package capnp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"

	"zombiezen.com/go/capnproto/internal/packed"
)

type buffer struct {
	Segment
	capTable
}

// NewBuffer creates an expanding single segment buffer. Creating new objects
// will expand the buffer. Data can be nil (or length 0 with some capacity) if
// creating a new session. If parsing an existing segment then data should be
// the segment contents and will not be copied.
func NewBuffer(data []byte) *Segment {
	if uint64(len(data)) > uint64(math.MaxUint32) {
		return nil
	}

	b := new(buffer)
	b.Message = b
	b.Data = data
	return &b.Segment
}

func (b *buffer) NewSegment(minsz Size) (*Segment, error) {
	if minsz < 4096 {
		minsz = 4096
	}
	n := len(b.Data)
	if uint64(n)+uint64(minsz) > uint64(math.MaxUint32) {
		return nil, errOverlarge
	}
	if nn := n + int(minsz); nn < cap(b.Data) {
		b.Data = b.Data[:nn]
	} else {
		newData := make([]byte, nn)
		copy(newData, b.Data)
		b.Data = newData
	}
	return &b.Segment, nil
}

func (b *buffer) Lookup(segid SegmentID) (*Segment, error) {
	if segid == 0 {
		return &b.Segment, nil
	} else {
		return nil, errInvalidSegment
	}
}

type multiBuffer struct {
	segments []*Segment
	capTable
}

// NewMultiBuffer creates a new multi segment message. Creating new objects
// will try and reuse the buffers available, but will create new ones if there
// is insufficient capacity. When parsing an existing message data should be
// the list of segments. The data buffers will not be copied.
func NewMultiBuffer(data [][]byte) *Segment {
	m := &multiBuffer{
		segments: make([]*Segment, len(data)),
	}
	for i, d := range data {
		m.segments[i] = &Segment{m, d, SegmentID(i), false}
	}
	if len(data) > 0 {
		return m.segments[0]
	}
	return &Segment{Message: m, Data: nil, Id: 0xFFFFFFFF, RootDone: false}
}

const (
	maxSegmentNumber = 1024
	maxTotalSize     = 1024 * 1024 * 1024
)

func (m *multiBuffer) NewSegment(minsz Size) (*Segment, error) {
	for _, s := range m.segments {
		if uint64(len(s.Data))+uint64(minsz) <= uint64(cap(s.Data)) {
			return s, nil
		}
	}

	if minsz < 4096 {
		minsz = 4096
	}
	s := &Segment{
		Message: m,
		Data:    make([]byte, 0, minsz),
		Id:      SegmentID(len(m.segments)),
	}
	m.segments = append(m.segments, s)
	return s, nil
}

func (m *multiBuffer) Lookup(segid SegmentID) (*Segment, error) {
	if uint(segid) < uint(len(m.segments)) {
		return m.segments[segid], nil
	} else {
		return nil, errInvalidSegment
	}
}

// ReadFromStream reads a non-packed serialized stream from r. buf is used to
// buffer the read contents, can be nil, and is provided so that the buffer
// can be reused between messages. The returned segment is the first segment
// read, which contains the root pointer.
//
// Warning about buf reuse:  It is safer to just pass nil for buf.
// When making multiple calls to ReadFromStream() with the same buf argument, you
// may overwrite the data in a previously returned Segment.
// The re-use of buf is an optimization for when you are actually
// done with any previously returned Segment which may have data still alive
// in buf.
//
func ReadFromStream(r io.Reader, buf *bytes.Buffer) (*Segment, error) {
	if buf == nil {
		buf = new(bytes.Buffer)
	} else {
		buf.Reset()
	}

	if _, err := io.CopyN(buf, r, 4); err != nil {
		return nil, err
	}

	if binary.LittleEndian.Uint32(buf.Bytes()[:]) >= uint32(maxSegmentNumber) {
		return nil, errTooMuchData
	}

	segnum := int(binary.LittleEndian.Uint32(buf.Bytes()[:]) + 1)
	hdrsz := 8*(segnum/2) + 4

	if _, err := io.CopyN(buf, r, int64(hdrsz)); err != nil {
		return nil, err
	}

	total := 0
	for i := 0; i < segnum; i++ {
		sz := binary.LittleEndian.Uint32(buf.Bytes()[4*i+4:])
		if uint64(total)+uint64(sz)*8 > uint64(maxTotalSize) {
			return nil, errTooMuchData
		}
		total += int(sz) * 8
	}

	if _, err := io.CopyN(buf, r, int64(total)); err != nil {
		return nil, err
	}

	hdrv := buf.Bytes()[4 : hdrsz+4]
	datav := buf.Bytes()[hdrsz+4:]

	if segnum == 1 {
		sz := int(binary.LittleEndian.Uint32(hdrv)) * 8
		return NewBuffer(datav[:sz]), nil
	}

	m := &multiBuffer{segments: make([]*Segment, segnum)}
	for i := 0; i < segnum; i++ {
		sz := int(binary.LittleEndian.Uint32(hdrv[4*i:])) * 8
		m.segments[i] = &Segment{
			Message: m,
			Data:    datav[:sz],
			Id:      SegmentID(i),
		}
		datav = datav[sz:]
	}

	return m.segments[0], nil
}

// ReadFromMemoryZeroCopy: like ReadFromStream, but reads a non-packed
// serialized stream that already resides in memory in the argument data.
// The returned segment is the first segment read, which contains
// the root pointer. The returned bytesRead says how many bytes were
// consumed from data in making seg. The caller should advance the
// data slice by doing data = data[bytesRead:] between successive calls
// to ReadFromMemoryZeroCopy().
func ReadFromMemoryZeroCopy(data []byte) (seg *Segment, bytesRead int64, err error) {

	if len(data) < 4 {
		return nil, 0, io.EOF
	}

	if binary.LittleEndian.Uint32(data[0:4]) >= uint32(maxSegmentNumber) {
		return nil, 0, errTooMuchData
	}

	segnum := int(binary.LittleEndian.Uint32(data[0:4]) + 1)
	hdrsz := 8*(segnum/2) + 4

	b := data[0:(hdrsz + 4)]

	total := 0
	for i := 0; i < segnum; i++ {
		sz := binary.LittleEndian.Uint32(b[4*i+4:])
		if uint64(total)+uint64(sz)*8 > uint64(maxTotalSize) {
			return nil, 0, errTooMuchData
		}
		total += int(sz) * 8
	}
	if total == 0 {
		return nil, 0, io.EOF
	}

	hdrv := data[4:(hdrsz + 4)]
	datav := data[hdrsz+4:]
	m := &multiBuffer{segments: make([]*Segment, segnum)}
	for i := 0; i < segnum; i++ {
		sz := int(binary.LittleEndian.Uint32(hdrv[4*i:])) * 8
		m.segments[i] = &Segment{
			Message: m,
			Data:    datav[:sz],
			Id:      SegmentID(i),
		}
		datav = datav[sz:]
	}

	return m.segments[0], int64(4 + hdrsz + total), nil
}

// ReadFromPackedStream reads a single message from the stream r in packed
// form returning the first segment. buf can be specified in order to reuse
// the buffer (or it is allocated each call if nil).
func ReadFromPackedStream(r io.Reader, buf *bytes.Buffer) (*Segment, error) {
	return ReadFromStream(packed.NewReader(r), buf)
}

func serialize(msg Message) []byte {
	// Compute buffer size.
	const (
		msgHeaderSize Size = 4
		segHeaderSize Size = 4
	)
	nsegs := numSegments(msg)
	hdrSize := int((msgHeaderSize + segHeaderSize.times(int32(nsegs))).padToWord())
	total := hdrSize
	for i := SegmentID(0); i < SegmentID(nsegs); i++ {
		s, _ := msg.Lookup(i)
		total += len(s.Data)
	}

	// Fill in buffer.
	buf := make([]byte, hdrSize, total)
	binary.LittleEndian.PutUint32(buf, nsegs-1)
	for i := SegmentID(0); i < SegmentID(nsegs); i++ {
		s, _ := msg.Lookup(i)
		binary.LittleEndian.PutUint32(buf[4*(i+1):], uint32(len(s.Data)/int(wordSize)))
	}
	for i := SegmentID(0); i < SegmentID(nsegs); i++ {
		s, _ := msg.Lookup(i)
		buf = append(buf, s.Data...)
	}
	return buf
}

// TODO(light): this smells.
func numSegments(msg Message) uint32 {
	n := uint32(1)
	for {
		if seg, _ := msg.Lookup(SegmentID(n)); seg == nil {
			return n
		}
		n++
	}
}

// WriteTo writes the message that the segment is part of to the
// provided stream in serialized form.
func (s *Segment) WriteTo(w io.Writer) (int64, error) {
	data := serialize(s.Message)
	n, err := w.Write(data)
	return int64(n), err
}

// WriteToPacked writes the message that the segment is part of to the
// provided stream in packed form.
func (s *Segment) WriteToPacked(w io.Writer) (int64, error) {
	data := serialize(s.Message)
	buf := make([]byte, 0, len(data))
	buf = packed.Pack(buf, data)
	n, err := w.Write(buf)
	return int64(n), err
}

type capTable []Client

func (tab capTable) CapTable() []Client {
	return []Client(tab)
}

func (tab *capTable) AddCap(c Client) CapabilityID {
	n := CapabilityID(len(*tab))
	*tab = append(*tab, c)
	return n
}

var (
	errBufferCall     = errors.New("capn: can't call on a memory buffer")
	errInvalidSegment = errors.New("capn: invalid segment id")
	errTooMuchData    = errors.New("capn: too much data in stream")
)
