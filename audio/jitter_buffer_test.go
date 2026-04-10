package audio

import "testing"

func TestJitterBuffer_ReordersDuringPrefill(t *testing.T) {
	jb := NewJitterBuffer(8, 2)

	jb.Push(101, []byte{0xB})
	jb.Push(100, []byte{0xA})

	p, ok := jb.Pop()
	if !ok || len(p) != 1 || p[0] != 0xA {
		t.Fatalf("expected seq 100 payload first, got ok=%v payload=%v", ok, p)
	}

	p, ok = jb.Pop()
	if !ok || len(p) != 1 || p[0] != 0xB {
		t.Fatalf("expected seq 101 payload second, got ok=%v payload=%v", ok, p)
	}
}

func TestJitterBuffer_DropsLatePacket(t *testing.T) {
	jb := NewJitterBuffer(8, 1)

	jb.Push(10, []byte{0x1})
	_, _ = jb.Pop() // consume seq 10, playSeq now 11
	_, _ = jb.Pop() // miss seq 11, playSeq now 12

	jb.Push(10, []byte{0x2}) // late packet, must be dropped

	p, ok := jb.Pop() // seq 12 expected (missing)
	if ok || p != nil {
		t.Fatalf("expected missing packet at seq 12, got ok=%v payload=%v", ok, p)
	}
}

func TestJitterBuffer_SequenceWrapAround(t *testing.T) {
	jb := NewJitterBuffer(8, 1)

	jb.Push(65535, []byte{0xF})
	p, ok := jb.Pop()
	if !ok || len(p) != 1 || p[0] != 0xF {
		t.Fatalf("expected seq 65535 payload, got ok=%v payload=%v", ok, p)
	}

	jb.Push(0, []byte{0x0})
	p, ok = jb.Pop()
	if !ok || len(p) != 1 || p[0] != 0x0 {
		t.Fatalf("expected wrapped seq 0 payload, got ok=%v payload=%v", ok, p)
	}
}
