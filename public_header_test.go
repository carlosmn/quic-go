package quic

import (
	"bytes"

	"github.com/lucas-clemente/quic-go/protocol"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Public Header", func() {
	Context("when parsing", func() {
		It("accepts a sample client header", func() {
			b := bytes.NewReader([]byte{0x09, 0xf6, 0x19, 0x86, 0x66, 0x9b, 0x9f, 0xfa, 0x4c, 0x51, 0x30, 0x33, 0x34, 0x01})
			hdr, err := ParsePublicHeader(b)
			Expect(err).ToNot(HaveOccurred())
			Expect(hdr.VersionFlag).To(BeTrue())
			Expect(hdr.ResetFlag).To(BeFalse())
			Expect(hdr.ConnectionID).To(Equal(protocol.ConnectionID(0x4cfa9f9b668619f6)))
			Expect(hdr.VersionNumber).To(Equal(protocol.Version34))
			Expect(hdr.PacketNumber).To(Equal(protocol.PacketNumber(1)))
			Expect(b.Len()).To(BeZero())
		})

		It("does not accept 0-byte connection ID", func() {
			b := bytes.NewReader([]byte{0x00, 0x01})
			_, err := ParsePublicHeader(b)
			Expect(err).To(MatchError(errReceivedTruncatedConnectionID))
		})

		It("rejects 0 as a connection ID", func() {
			b := bytes.NewReader([]byte{0x09, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x51, 0x30, 0x33, 0x30, 0x01})
			_, err := ParsePublicHeader(b)
			Expect(err).To(MatchError(errInvalidConnectionID))
		})

		It("accepts 1-byte packet numbers", func() {
			b := bytes.NewReader([]byte{0x08, 0xf6, 0x19, 0x86, 0x66, 0x9b, 0x9f, 0xfa, 0x4c, 0xde})
			hdr, err := ParsePublicHeader(b)
			Expect(err).ToNot(HaveOccurred())
			Expect(hdr.PacketNumber).To(Equal(protocol.PacketNumber(0xde)))
			Expect(b.Len()).To(BeZero())
		})

		It("accepts 2-byte packet numbers", func() {
			b := bytes.NewReader([]byte{0x18, 0xf6, 0x19, 0x86, 0x66, 0x9b, 0x9f, 0xfa, 0x4c, 0xde, 0xca})
			hdr, err := ParsePublicHeader(b)
			Expect(err).ToNot(HaveOccurred())
			Expect(hdr.PacketNumber).To(Equal(protocol.PacketNumber(0xcade)))
			Expect(b.Len()).To(BeZero())
		})

		It("accepts 4-byte packet numbers", func() {
			b := bytes.NewReader([]byte{0x28, 0xf6, 0x19, 0x86, 0x66, 0x9b, 0x9f, 0xfa, 0x4c, 0xad, 0xfb, 0xca, 0xde})
			hdr, err := ParsePublicHeader(b)
			Expect(err).ToNot(HaveOccurred())
			Expect(hdr.PacketNumber).To(Equal(protocol.PacketNumber(0xdecafbad)))
			Expect(b.Len()).To(BeZero())
		})

		It("accepts 6-byte packet numbers", func() {
			b := bytes.NewReader([]byte{0x38, 0xf6, 0x19, 0x86, 0x66, 0x9b, 0x9f, 0xfa, 0x4c, 0x23, 0x42, 0xad, 0xfb, 0xca, 0xde})
			hdr, err := ParsePublicHeader(b)
			Expect(err).ToNot(HaveOccurred())
			Expect(hdr.PacketNumber).To(Equal(protocol.PacketNumber(0xdecafbad4223)))
			Expect(b.Len()).To(BeZero())
		})

		PIt("rejects diversification nonces sent by the client", func() {
			b := bytes.NewReader([]byte{0x0c, 0xf6, 0x19, 0x86, 0x66, 0x9b, 0x9f, 0xfa, 0x4c,
				0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1,
				0x01,
			})
			_, err := ParsePublicHeader(b)
			Expect(err).To(MatchError("diversification nonces should only be sent by servers"))
		})
	})

	Context("when writing", func() {
		It("writes a sample header", func() {
			b := &bytes.Buffer{}
			hdr := PublicHeader{
				ConnectionID:    0x4cfa9f9b668619f6,
				PacketNumber:    2,
				PacketNumberLen: protocol.PacketNumberLen6,
			}
			hdr.Write(b, protocol.Version35)
			Expect(b.Bytes()).To(Equal([]byte{0x38, 0xf6, 0x19, 0x86, 0x66, 0x9b, 0x9f, 0xfa, 0x4c, 2, 0, 0, 0, 0, 0}))
		})

		It("sets the Version Flag", func() {
			b := &bytes.Buffer{}
			hdr := PublicHeader{
				VersionFlag:     true,
				ConnectionID:    0x4cfa9f9b668619f6,
				PacketNumber:    2,
				PacketNumberLen: protocol.PacketNumberLen6,
			}
			hdr.Write(b, protocol.VersionWhatever)
			// must be the first assertion
			Expect(b.Len()).To(Equal(1 + 8)) // 1 FlagByte + 8 ConnectionID
			firstByte, _ := b.ReadByte()
			Expect(firstByte & 0x01).To(Equal(uint8(1)))
		})

		It("sets the Reset Flag", func() {
			b := &bytes.Buffer{}
			hdr := PublicHeader{
				ResetFlag:       true,
				ConnectionID:    0x4cfa9f9b668619f6,
				PacketNumber:    2,
				PacketNumberLen: protocol.PacketNumberLen6,
			}
			hdr.Write(b, protocol.VersionWhatever)
			// must be the first assertion
			Expect(b.Len()).To(Equal(1 + 8)) // 1 FlagByte + 8 ConnectionID
			firstByte, _ := b.ReadByte()
			Expect((firstByte & 0x02) >> 1).To(Equal(uint8(1)))
		})

		It("throws an error if both Reset Flag and Version Flag are set", func() {
			b := &bytes.Buffer{}
			hdr := PublicHeader{
				VersionFlag:     true,
				ResetFlag:       true,
				ConnectionID:    0x4cfa9f9b668619f6,
				PacketNumber:    2,
				PacketNumberLen: protocol.PacketNumberLen6,
			}
			err := hdr.Write(b, protocol.VersionWhatever)
			Expect(err).To(MatchError(errResetAndVersionFlagSet))
		})

		It("truncates the connection ID", func() {
			b := &bytes.Buffer{}
			hdr := PublicHeader{
				ConnectionID:         0x4cfa9f9b668619f6,
				TruncateConnectionID: true,
				PacketNumberLen:      protocol.PacketNumberLen6,
				PacketNumber:         1,
			}
			err := hdr.Write(b, protocol.VersionWhatever)
			Expect(err).ToNot(HaveOccurred())
			Expect(b.Bytes()).To(Equal([]byte{0x30, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0}))
		})

		It("writes proper v33 packets", func() {
			b := &bytes.Buffer{}
			hdr := PublicHeader{
				ConnectionID:    0x4cfa9f9b668619f6,
				PacketNumber:    1,
				PacketNumberLen: protocol.PacketNumberLen1,
			}
			err := hdr.Write(b, protocol.Version35)
			Expect(err).ToNot(HaveOccurred())
			Expect(b.Bytes()).To(Equal([]byte{0x08, 0xf6, 0x19, 0x86, 0x66, 0x9b, 0x9f, 0xfa, 0x4c, 0x01}))
		})

		It("writes diversification nonces", func() {
			b := &bytes.Buffer{}
			hdr := PublicHeader{
				ConnectionID:         0x4cfa9f9b668619f6,
				PacketNumber:         1,
				PacketNumberLen:      protocol.PacketNumberLen1,
				DiversificationNonce: bytes.Repeat([]byte{1}, 32),
			}
			err := hdr.Write(b, protocol.Version35)
			Expect(err).ToNot(HaveOccurred())
			Expect(b.Bytes()).To(Equal([]byte{
				0x0c, 0xf6, 0x19, 0x86, 0x66, 0x9b, 0x9f, 0xfa, 0x4c,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				0x01,
			}))
		})

		Context("GetLength", func() {
			It("errors when calling GetLength for Version Negotiation packets", func() {
				hdr := PublicHeader{VersionFlag: true}
				_, err := hdr.GetLength()
				Expect(err).To(MatchError(errGetLengthOnlyForRegularPackets))
			})

			It("errors when calling GetLength for Public Reset packets", func() {
				hdr := PublicHeader{ResetFlag: true}
				_, err := hdr.GetLength()
				Expect(err).To(MatchError(errGetLengthOnlyForRegularPackets))
			})

			It("errors when PacketNumberLen is not set", func() {
				hdr := PublicHeader{
					ConnectionID: 0x4cfa9f9b668619f6,
					PacketNumber: 0xDECAFBAD,
				}
				_, err := hdr.GetLength()
				Expect(err).To(MatchError(errPacketNumberLenNotSet))
			})

			It("gets the length of a packet with longest packet number length and connectionID", func() {
				hdr := PublicHeader{
					ConnectionID:    0x4cfa9f9b668619f6,
					PacketNumber:    0xDECAFBAD,
					PacketNumberLen: protocol.PacketNumberLen6,
				}
				length, err := hdr.GetLength()
				Expect(err).ToNot(HaveOccurred())
				Expect(length).To(Equal(protocol.ByteCount(1 + 8 + 6))) // 1 byte public flag, 8 bytes connectionID, and packet number
			})

			It("gets the length of a packet with longest packet number length and truncated connectionID", func() {
				hdr := PublicHeader{
					ConnectionID:         0x4cfa9f9b668619f6,
					TruncateConnectionID: true,
					PacketNumber:         0xDECAFBAD,
					PacketNumberLen:      protocol.PacketNumberLen6,
				}
				length, err := hdr.GetLength()
				Expect(err).ToNot(HaveOccurred())
				Expect(length).To(Equal(protocol.ByteCount(1 + 6))) // 1 byte public flag, and packet number
			})

			It("gets the length of a packet 2 byte packet number length ", func() {
				hdr := PublicHeader{
					ConnectionID:    0x4cfa9f9b668619f6,
					PacketNumber:    0xDECAFBAD,
					PacketNumberLen: protocol.PacketNumberLen2,
				}
				length, err := hdr.GetLength()
				Expect(err).ToNot(HaveOccurred())
				Expect(length).To(Equal(protocol.ByteCount(1 + 8 + 2))) // 1 byte public flag, 8 byte connectionID, and packet number
			})

			It("works with diversification nonce", func() {
				hdr := PublicHeader{
					DiversificationNonce: []byte("foo"),
					PacketNumberLen:      protocol.PacketNumberLen1,
				}
				length, err := hdr.GetLength()
				Expect(err).NotTo(HaveOccurred())
				Expect(length).To(Equal(protocol.ByteCount(1 + 8 + 3 + 1)))
			})
		})

		Context("packet number length", func() {
			It("doesn't write a header if the packet number length is not set", func() {
				b := &bytes.Buffer{}
				hdr := PublicHeader{
					ConnectionID: 0x4cfa9f9b668619f6,
					PacketNumber: 0xDECAFBAD,
				}
				err := hdr.Write(b, protocol.VersionWhatever)
				Expect(err).To(MatchError(errPacketNumberLenNotSet))
			})

			It("writes a header with a 1-byte packet number", func() {
				b := &bytes.Buffer{}
				hdr := PublicHeader{
					ConnectionID:    0x4cfa9f9b668619f6,
					PacketNumber:    0xDECAFBAD,
					PacketNumberLen: protocol.PacketNumberLen1,
				}
				err := hdr.Write(b, protocol.VersionWhatever)
				Expect(err).ToNot(HaveOccurred())
				Expect(b.Bytes()).To(Equal([]byte{0x08, 0xf6, 0x19, 0x86, 0x66, 0x9b, 0x9f, 0xfa, 0x4c, 0xAD}))
			})

			It("writes a header with a 2-byte packet number", func() {
				b := &bytes.Buffer{}
				hdr := PublicHeader{
					ConnectionID:    0x4cfa9f9b668619f6,
					PacketNumber:    0xDECAFBAD,
					PacketNumberLen: protocol.PacketNumberLen2,
				}
				err := hdr.Write(b, protocol.VersionWhatever)
				Expect(err).ToNot(HaveOccurred())
				Expect(b.Bytes()).To(Equal([]byte{0x18, 0xf6, 0x19, 0x86, 0x66, 0x9b, 0x9f, 0xfa, 0x4c, 0xAD, 0xFB}))
			})

			It("writes a header with a 4-byte packet number", func() {
				b := &bytes.Buffer{}
				hdr := PublicHeader{
					ConnectionID:    0x4cfa9f9b668619f6,
					PacketNumber:    0x13DECAFBAD,
					PacketNumberLen: protocol.PacketNumberLen4,
				}
				err := hdr.Write(b, protocol.VersionWhatever)
				Expect(err).ToNot(HaveOccurred())
				Expect(b.Bytes()).To(Equal([]byte{0x28, 0xf6, 0x19, 0x86, 0x66, 0x9b, 0x9f, 0xfa, 0x4c, 0xAD, 0xFB, 0xCA, 0xDE}))
			})

			It("writes a header with a 6-byte packet number", func() {
				b := &bytes.Buffer{}
				hdr := PublicHeader{
					ConnectionID:    0x4cfa9f9b668619f6,
					PacketNumber:    0xBE1337DECAFBAD,
					PacketNumberLen: protocol.PacketNumberLen6,
				}
				err := hdr.Write(b, protocol.VersionWhatever)
				Expect(err).ToNot(HaveOccurred())
				Expect(b.Bytes()).To(Equal([]byte{0x38, 0xf6, 0x19, 0x86, 0x66, 0x9b, 0x9f, 0xfa, 0x4c, 0xAD, 0xFB, 0xCA, 0xDE, 0x37, 0x13}))
			})
		})
	})
})
