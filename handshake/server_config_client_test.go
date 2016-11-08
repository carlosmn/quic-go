package handshake

import (
	"bytes"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server Config", func() {
	var tagMap map[Tag][]byte

	BeforeEach(func() {
		// This tagMap can be passed to parseValues and doesn't cause any errors
		tagMap = make(map[Tag][]byte)
		tagMap[TagSCID] = bytes.Repeat([]byte{'F'}, 16)
		tagMap[TagKEXS] = []byte("C255")
		tagMap[TagAEAD] = []byte("AESG")
		tagMap[TagPUBS] = bytes.Repeat([]byte{0}, 35)
		tagMap[TagOBIT] = bytes.Repeat([]byte{0}, 8)
		tagMap[TagEXPY] = bytes.Repeat([]byte{0}, 8)
	})

	Context("parsing the server config", func() {
		It("rejects a handshake message with the wrong message tag", func() {
			var serverConfig bytes.Buffer
			WriteHandshakeMessage(&serverConfig, TagCHLO, make(map[Tag][]byte))
			_, err := parseServerConfig(serverConfig.Bytes())
			Expect(err).To(MatchError(errMessageNotServerConfig))
		})

		It("errors on invalid handshake messages", func() {
			var serverConfig bytes.Buffer
			WriteHandshakeMessage(&serverConfig, TagSCFG, make(map[Tag][]byte))
			_, err := parseServerConfig(serverConfig.Bytes()[:serverConfig.Len()-2])
			Expect(err).To(MatchError("unexpected EOF"))
		})

		It("passes on errors encountered when reading the TagMap", func() {
			var serverConfig bytes.Buffer
			WriteHandshakeMessage(&serverConfig, TagSCFG, make(map[Tag][]byte))
			_, err := parseServerConfig(serverConfig.Bytes())
			Expect(err).To(MatchError("CryptoMessageParameterNotFound: SCID"))
		})

		It("reads an example Handshake Message", func() {
			var serverConfig bytes.Buffer
			WriteHandshakeMessage(&serverConfig, TagSCFG, tagMap)
			scfg, err := parseServerConfig(serverConfig.Bytes())
			Expect(err).ToNot(HaveOccurred())
			Expect(scfg.ID).To(Equal(tagMap[TagSCID]))
			Expect(scfg.obit).To(Equal(tagMap[TagOBIT]))
		})
	})

	Context("Reading values fromt the TagMap", func() {
		var scfg *serverConfigClient

		BeforeEach(func() {
			scfg = &serverConfigClient{}
		})

		Context("ServerConfig ID", func() {
			It("parses the ServerConfig ID", func() {
				id := []byte{0xb2, 0xa4, 0xbb, 0x8f, 0xf6, 0x51, 0x28, 0xfd, 0x4d, 0xf7, 0xb3, 0x9a, 0x91, 0xe7, 0x91, 0xfb}
				tagMap[TagSCID] = id
				err := scfg.parseValues(tagMap)
				Expect(err).ToNot(HaveOccurred())
				Expect(scfg.ID).To(Equal(id))
			})

			It("errors if the ServerConfig ID is missing", func() {
				delete(tagMap, TagSCID)
				err := scfg.parseValues(tagMap)
				Expect(err).To(MatchError("CryptoMessageParameterNotFound: SCID"))
			})

			It("rejects ServerConfig IDs that have the wrong length", func() {
				tagMap[TagSCID] = bytes.Repeat([]byte{'F'}, 17) // 1 byte too long
				err := scfg.parseValues(tagMap)
				Expect(err).To(MatchError("CryptoInvalidValueLength: SCID"))
			})
		})

		Context("KEXS", func() {
			It("rejects KEXS values that have the wrong length", func() {
				tagMap[TagKEXS] = bytes.Repeat([]byte{'F'}, 5) // 1 byte too long
				err := scfg.parseValues(tagMap)
				Expect(err).To(MatchError("CryptoInvalidValueLength: KEXS"))
			})

			It("rejects KEXS values other than C255", func() {
				tagMap[TagKEXS] = []byte("P256")
				err := scfg.parseValues(tagMap)
				Expect(err).To(MatchError("CryptoNoSupport: KEXS"))
			})

			It("errors if the KEXS is missing", func() {
				delete(tagMap, TagKEXS)
				err := scfg.parseValues(tagMap)
				Expect(err).To(MatchError("CryptoMessageParameterNotFound: KEXS"))
			})
		})

		Context("AEAD", func() {
			It("rejects AEAD values that have the wrong length", func() {
				tagMap[TagAEAD] = bytes.Repeat([]byte{'F'}, 5) // 1 byte too long
				err := scfg.parseValues(tagMap)
				Expect(err).To(MatchError("CryptoInvalidValueLength: AEAD"))
			})

			It("rejects AEAD values other than AESG", func() {
				tagMap[TagAEAD] = []byte("S20P")
				err := scfg.parseValues(tagMap)
				Expect(err).To(MatchError("CryptoNoSupport: AEAD"))
			})

			It("recognizes AESG in the list of AEADs, at the first position", func() {
				tagMap[TagAEAD] = []byte("AESGS20P")
				err := scfg.parseValues(tagMap)
				Expect(err).ToNot(HaveOccurred())
			})

			It("recognizes AESG in the list of AEADs, not at the first position", func() {
				tagMap[TagAEAD] = []byte("S20PAESG")
				err := scfg.parseValues(tagMap)
				Expect(err).ToNot(HaveOccurred())
			})

			It("errors if the AEAD is missing", func() {
				delete(tagMap, TagAEAD)
				err := scfg.parseValues(tagMap)
				Expect(err).To(MatchError("CryptoMessageParameterNotFound: AEAD"))
			})
		})

		Context("PUBS", func() {
			It("rejects PUBS values that have the wrong length", func() {
				tagMap[TagPUBS] = bytes.Repeat([]byte{'F'}, 100) // completely wrong length
				err := scfg.parseValues(tagMap)
				Expect(err).To(MatchError("CryptoInvalidValueLength: PUBS"))
			})

			It("errors if the PUBS is missing", func() {
				delete(tagMap, TagPUBS)
				err := scfg.parseValues(tagMap)
				Expect(err).To(MatchError("CryptoMessageParameterNotFound: PUBS"))
			})
		})

		Context("OBIT", func() {
			It("parses the OBIT value", func() {
				obit := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8}
				tagMap[TagOBIT] = obit
				err := scfg.parseValues(tagMap)
				Expect(err).ToNot(HaveOccurred())
				Expect(scfg.obit).To(Equal(obit))
			})

			It("errors if the OBIT is missing", func() {
				delete(tagMap, TagOBIT)
				err := scfg.parseValues(tagMap)
				Expect(err).To(MatchError("CryptoMessageParameterNotFound: OBIT"))
			})

			It("rejets OBIT values that have the wrong length", func() {
				tagMap[TagOBIT] = bytes.Repeat([]byte{'F'}, 7) // 1 byte too short
				err := scfg.parseValues(tagMap)
				Expect(err).To(MatchError("CryptoInvalidValueLength: OBIT"))
			})
		})

		Context("EXPY", func() {
			It("parses the expiry date", func() {
				tagMap[TagEXPY] = []byte{0xdc, 0x89, 0x0e, 0x59, 0, 0, 0, 0} // UNIX Timestamp 0x590e89dc = 1494125020
				err := scfg.parseValues(tagMap)
				Expect(err).ToNot(HaveOccurred())
				year, month, day := scfg.expiry.Date()
				Expect(year).To(Equal(2017))
				Expect(month).To(Equal(time.Month(5)))
				Expect(day).To(Equal(7))
			})

			It("errors if the EXPY is missing", func() {
				delete(tagMap, TagEXPY)
				err := scfg.parseValues(tagMap)
				Expect(err).To(MatchError("CryptoMessageParameterNotFound: EXPY"))
			})

			It("rejects EXPY values that have the wrong length", func() {
				tagMap[TagEXPY] = bytes.Repeat([]byte{'F'}, 9) // 1 byte too long
				err := scfg.parseValues(tagMap)
				Expect(err).To(MatchError("CryptoInvalidValueLength: EXPY"))
			})
		})
	})
})