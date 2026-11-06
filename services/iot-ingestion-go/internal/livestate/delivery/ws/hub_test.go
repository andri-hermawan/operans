package ws

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func received(s *Subscriber) []Envelope {
	var out []Envelope
	for {
		select {
		case b := <-s.C():
			var e Envelope
			json.Unmarshal(b, &e)
			out = append(out, e)
		default:
			return out
		}
	}
}

func TestHubPerProject(t *testing.T) {
	h := NewHub(10, 8)
	projectA, projectB := uuid.New(), uuid.New()
	subA, _ := h.Subscribe(&projectA)
	subB, _ := h.Subscribe(&projectB)
	subSA, _ := h.Subscribe(nil) // superadmin

	h.Publish(&projectA, TypeEquipmentState, map[string]string{"unit": "A1"})
	h.Publish(&projectB, TypeEquipmentState, map[string]string{"unit": "B1"})

	if n := len(received(subA)); n != 1 {
		t.Fatalf("project A seharusnya menerima 1 pesan, dapat %d", n)
	}
	if n := len(received(subB)); n != 1 {
		t.Fatalf("project B seharusnya menerima 1 pesan, dapat %d", n)
	}
	if n := len(received(subSA)); n != 2 {
		t.Fatalf("superadmin seharusnya menerima 2 pesan, dapat %d", n)
	}
}

func TestHubPelangganLambatDiputus(t *testing.T) {
	h := NewHub(10, 2) // antrean hanya 2 pesan
	p := uuid.New()
	lambat, _ := h.Subscribe(&p)

	for i := 0; i < 3; i++ { // pesan ke-3 tidak muat
		h.Publish(&p, TypeEquipmentState, i)
	}
	select {
	case <-lambat.Done():
		if lambat.Reason() != ReasonSlow {
			t.Fatalf("alasan = %q", lambat.Reason())
		}
	default:
		t.Fatal("pelanggan yang antreannya penuh seharusnya diputus")
	}
}

func TestHubBatasDanClose(t *testing.T) {
	h := NewHub(1, 8)
	p := uuid.New()
	s, err := h.Subscribe(&p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Subscribe(&p); err != ErrFull {
		t.Fatalf("koneksi ke-2 seharusnya ditolak, err = %v", err)
	}
	h.Close()
	select {
	case <-s.Done():
	default:
		t.Fatal("Close seharusnya memutus semua pelanggan")
	}
	h.Unsubscribe(s)
	if h.Count() != 0 {
		t.Fatal("pelanggan seharusnya sudah terhapus")
	}
}
