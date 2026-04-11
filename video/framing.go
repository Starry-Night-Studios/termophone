package video

import (
	"bufio"
	"encoding/binary"
	"io"
	"log"
)

func writeFull(w io.Writer, p []byte) error {
	for len(p) > 0 {
		n, err := w.Write(p)
		if err != nil {
			return err
		}
		p = p[n:]
	}
	return nil
}

func findStartCode(data []byte, from int) int {
	if from < 0 {
		from = 0
	}
	for i := from; i+3 < len(data); i++ {
		if data[i] == 0 && data[i+1] == 0 {
			if data[i+2] == 1 {
				return i
			}
			if i+3 < len(data) && data[i+2] == 0 && data[i+3] == 1 {
				return i
			}
		}
	}
	return -1
}

func startCodeLen(data []byte, at int) int {
	if at+3 < len(data) && data[at] == 0 && data[at+1] == 0 {
		if data[at+2] == 1 {
			return 3
		}
		if data[at+2] == 0 && data[at+3] == 1 {
			return 4
		}
	}
	return 0
}

// videoWriter parses Annex B NAL units and sends each one with a 4-byte big-endian length prefix.
func videoWriter(r io.Reader, w io.Writer) error {
	br := bufio.NewReaderSize(r, 64*1024)
	buf := make([]byte, 0, 256*1024)
	tmp := make([]byte, 64*1024)
	hdr := make([]byte, 4)

	sendNAL := func(nal []byte) error {
		if len(nal) == 0 {
			return nil
		}
		binary.BigEndian.PutUint32(hdr, uint32(len(nal)))
		if err := writeFull(w, hdr); err != nil {
			return err
		}
		return writeFull(w, nal)
	}

	for {
		n, err := br.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)

			for {
				first := findStartCode(buf, 0)
				if first == -1 {
					break
				}

				sc := startCodeLen(buf, first)
				if sc == 0 {
					break
				}

				nalStart := first + sc
				next := findStartCode(buf, nalStart)
				if next == -1 {
					if first > 0 {
						buf = buf[first:]
					}
					break
				}

				if err := sendNAL(buf[nalStart:next]); err != nil {
					return err
				}
				buf = buf[next:]
			}

			// Guard against unbounded growth if start codes are missing/corrupt.
			if len(buf) > 4*1024*1024 {
				return io.ErrUnexpectedEOF
			}
		}

		if err != nil {
			if err == io.EOF {
				last := findStartCode(buf, 0)
				if last != -1 {
					sc := startCodeLen(buf, last)
					if sc > 0 {
						if sendErr := sendNAL(buf[last+sc:]); sendErr != nil {
							return sendErr
						}
					}
				}
				return nil
			}
			return err
		}
	}
}

// videoReader decodes 4-byte big-endian length-prefixed NAL units back into Annex B byte-stream format.
func videoReader(r io.Reader, w io.Writer) error {
	hdr := make([]byte, 4)
	for {
		if _, err := io.ReadFull(r, hdr); err != nil {
			if err != io.EOF {
				log.Println("video read error:", err)
			}
			return err
		}

		length := binary.BigEndian.Uint32(hdr)
		if length == 0 || length > 4*1024*1024 {
			return io.ErrUnexpectedEOF
		}

		nal := make([]byte, length)
		if _, err := io.ReadFull(r, nal); err != nil {
			log.Println("video NAL read error:", err)
			return err
		}

		if err := writeFull(w, h264StartCode); err != nil {
			return err
		}
		if err := writeFull(w, nal); err != nil {
			return err
		}
	}
}
