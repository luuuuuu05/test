package udp

import (
	"log"
	"net"
	"time"

	"mrflag/internal/room"
)

type Server struct {
	addr string
	mgr  *room.Manager
}

type packetStat struct {
	received  uint64
	relayed   uint64
	dropped   uint64
	lastLogAt time.Time
}

func NewServer(addr string, mgr *room.Manager) *Server {
	return &Server{addr: addr, mgr: mgr}
}

func (s *Server) ListenAndServe() error {
	udpAddr, err := net.ResolveUDPAddr("udp", s.addr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Printf("udp listening on %s", s.addr)

	buf := make([]byte, 2048)
	stats := map[string]*packetStat{}
	for {
		n, from, err := conn.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		pkt, err := DecodePosPacket(buf[:n])
		if err != nil {
			log.Printf("drop invalid udp packet from %s: %v", from, err)
			continue
		}
		key := pkt.RoomID + "/" + pkt.PlayerID
		stat := stats[key]
		if stat == nil {
			stat = &packetStat{}
			stats[key] = stat
		}
		stat.received++
		relays := s.mgr.HandleUDPPacket(pkt, from)
		if len(relays) == 0 {
			stat.dropped++
			s.maybeLogUDPStat("udp recv no relay", pkt.RoomID, pkt.PlayerID, pkt.Seq, from, 0, stat)
			continue
		}
		payload := EncodeRelayPacket(relays[0])
		recipients := s.mgr.UDPRecipients(pkt.RoomID, pkt.PlayerID)
		sent := 0
		failed := 0
		for _, to := range recipients {
			if _, err := conn.WriteToUDP(payload, to); err != nil {
				failed++
				log.Printf("udp send failed room=%s from_player=%s to=%s seq=%d err=%v", pkt.RoomID, pkt.PlayerID, to, pkt.Seq, err)
				continue
			}
			sent++
		}
		stat.relayed += uint64(sent)
		if sent == 0 {
			stat.dropped++
			s.maybeLogUDPStat("udp relay no recipient sent", pkt.RoomID, pkt.PlayerID, pkt.Seq, from, sent, stat)
			continue
		}
		if failed > 0 {
			log.Printf("udp relay partial room=%s player=%s seq=%d sent=%d failed=%d", pkt.RoomID, pkt.PlayerID, pkt.Seq, sent, failed)
		}
		s.maybeLogUDPStat("udp relay ok", pkt.RoomID, pkt.PlayerID, pkt.Seq, from, sent, stat)
	}
}

func (s *Server) maybeLogUDPStat(prefix, roomID, playerID string, seq uint32, from *net.UDPAddr, recipients int, stat *packetStat) {
	now := time.Now()
	if stat.received <= 3 || stat.received%30 == 0 || now.Sub(stat.lastLogAt) >= 2*time.Second {
		log.Printf("%s room=%s player=%s seq=%d from=%s recipients=%d received=%d relayed=%d dropped=%d", prefix, roomID, playerID, seq, from, recipients, stat.received, stat.relayed, stat.dropped)
		stat.lastLogAt = now
	}
}
