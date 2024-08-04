package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// Great info from:
// https://www.ietf.org/rfc/rfc1035.txt
// Page 24 onwards

type Header struct {
	id uint16
	/*
	 * Within Packed:
	 *     QR     - 1 bit
	 *     Opcode - 4
	 *     AA     - 1
	 *     TC     - 1
	 *     RD     - 1
	 *     RA     - 1
	 *     Z      - 3
	 *     RCODE  - 4
	 */
	packed   uint16
	qdcount  uint16
	ancount  uint16
	nscount  uint16
	artcount uint16
}

type Message struct {
	header      Header
	questions   []Question
	answers     []Answer
	authorities []Authority
	additional  []Additional
}

type Question struct {
	// This should get stored as
	// 1 byte of length
	// n bytes of data
	// a null byte terminator
	qname  string
	qtype  uint16
	qclass uint16
}

type Answer struct {
	// Same as qname above
	name     string
	kind     uint16 // aka type, a keyword in Go ;)
	class    uint16
	ttl      uint32
	rdlength uint16
	/*
	 * If the TYPE is 0x0001 for A records, then this is the IP address ( octets).
	 * If the type is 0x0005 for CNAMEs, then this is the name of the alias.
	 * If the type is 0x0002 for name servers, then this is the name of the server.
	 * Finally if the type is 0x000f for mail servers, the format is
	 *     PREFERENCE
	 *     EXCHANCE
	 * where PREFERENCE is a 16 bit integer which specifies the preference of this mail server, and EXCHANGE is a domain name stored in the same format as QNAMEs
	 */
	rdata []byte
}

type Authority struct {
	// TODO
}
type Additional struct {
	// TODO
}

func main() {
	return
}

//***************************************************************************************** Helpers
/*
                                   1  1  1  1  1  1
     0  1  2  3  4  5  6  7  8  9  0  1  2  3  4  5
   +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
   |QR|   Opcode  |AA|TC|RD|RA|   Z    |   RCODE   |

   - We want to put ra, z, and rcode first, and the rest second, because we are doing big endian
   - We shall do it by adding and subtracting 8 (ew)
*/
func packed(qr, opcode, aa, tc, rd, ra, z, rcode uint8) (out uint16) {
	out |= uint16(qr) << (0 + 8)
	out |= uint16(opcode) << (1 + 8)
	out |= uint16(aa) << (5 + 8)
	out |= uint16(tc) << (6 + 8)
	out |= uint16(rd) << (7 + 8)
	out |= uint16(ra) << 7
	out |= uint16(z) << 4
	out |= uint16(rcode) << 0
	return out
}

func unpacked(in uint16) (qr, opcode, aa, tc, rd, ra, z, rcode uint8) {
	qr = uint8((in & 0b0000_0000_0000_0001) >> 0)
	opcode = uint8((in & 0b0000_0000_0001_1110) >> 1)
	aa = uint8((in & 0b0000_0000_0010_0000) >> 5)
	tc = uint8((in & 0b0000_0000_0100_0000) >> 6)
	rd = uint8((in & 0b0000_0000_1000_0000) >> 7)
	ra = uint8((in & 0b0000_0001_0000_0000) >> 8)
	z = uint8((in & 0b0000_1110_0000_0000) >> 9)
	rcode = uint8((in & 0b1111_0000_0000_0000) >> 12)
	return qr, opcode, aa, tc, rd, ra, z, rcode
}

//*************************************************************************************** Top-Level
func Serialize(response Message) (data []byte, err error) {
	namemap := map[uint16]string{}
	offset := uint16(0)
	buf := new(bytes.Buffer)

	n, err := serialHeader(buf, response.header)
	if err != nil {
		return nil, fmt.Errorf("Failed to serialize response header: %s", err)
	}
	offset += n

	for _, question := range response.questions {
		written, err := serialQuestion(buf, namemap, offset, &question)
		if err != nil {
			return nil, fmt.Errorf("Failed to serialize question: %s", err)
		}
		offset += written
	}
	for _, answer := range response.answers {
		written, err := serialAnswer(buf, namemap, offset, &answer)
		if err != nil {
			return nil, fmt.Errorf("Failed to serialize answer: %s", err)
		}
		offset += written
	}
	for _, authority := range response.authorities {
		if err = serialAuthority(buf, &authority); err != nil {
			return nil, fmt.Errorf("Failed to serialize authority: %s", err)
		}
	}
	for _, additional := range response.additional {
		if err = serialAdditional(buf, &additional); err != nil {
			return nil, fmt.Errorf("Failed to serialize additional: %s", err)
		}
	}

	return buf.Bytes(), nil
}

func Deserialize(data []byte) (request Message, err error) {
	namemap := map[uint16]string{}
	offset := uint16(0)
	r := bytes.NewReader(data)

	read, err := deserialHeader(r, &request.header)
	if err != nil {
		return request, fmt.Errorf("Failed to deserialize request header: %s", err)
	}
	offset += read

	for i := 0; i < int(request.header.qdcount); i++ {
		question := Question{}
		read, err := deserialQuestion(r, namemap, offset, &question)
		if err != nil {
			return request, fmt.Errorf("Failed to deserialize question #%d: %s", i, err)
		}
		offset += read
		request.questions = append(request.questions, question)
	}

	for _, answer := range request.answers {
		read, err := deserialAnswer(r, namemap, offset, &answer)
		if err != nil {
			return request, fmt.Errorf("Failed to deserialize answer: %s", err)
		}
		offset += read
		request.answers = append(request.answers, answer)
	}
	for _, authority := range request.authorities {
		if err = deserialAuthority(r, &authority); err != nil {
			return request, fmt.Errorf("Failed to deserialize authority: %s", err)
		}
	}
	for _, additional := range request.additional {
		if err = deserialAdditional(r, &additional); err != nil {
			return request, fmt.Errorf("Failed to deserialize additional: %s", err)
		}
	}

	return request, nil
}

//****************************************************************************************** Header
func deserialHeader(r *bytes.Reader, header *Header) (read uint16, err error) {
	if err = binary.Read(r, binary.BigEndian, &header.id); err != nil {
		return read, fmt.Errorf("Failed to read id: %s", err)
	}
	read += 2
	if err = binary.Read(r, binary.BigEndian, &header.packed); err != nil {
		return read, fmt.Errorf("Failed to read packed: %s", err)
	}
	read += 2
	if err = binary.Read(r, binary.BigEndian, &header.qdcount); err != nil {
		return read, fmt.Errorf("Failed to read qdcount: %s", err)
	}
	read += 2
	if err = binary.Read(r, binary.BigEndian, &header.ancount); err != nil {
		return read, fmt.Errorf("Failed to read ancount: %s", err)
	}
	read += 2
	if err = binary.Read(r, binary.BigEndian, &header.nscount); err != nil {
		return read, fmt.Errorf("Failed to read nscount: %s", err)
	}
	read += 2
	if err = binary.Read(r, binary.BigEndian, &header.artcount); err != nil {
		return read, fmt.Errorf("Failed to read artcount: %s", err)
	}
	read += 2
	return read, nil
}

func serialHeader(buf *bytes.Buffer, header Header) (n uint16, err error) {
	if err = binary.Write(buf, binary.BigEndian, header.id); err != nil {
		return n, fmt.Errorf("Failed to write id: %s", err)
	}
	n += 2
	if err = binary.Write(buf, binary.BigEndian, header.packed); err != nil {
		return n, fmt.Errorf("Failed to write packed: %s", err)
	}
	n += 2
	if err = binary.Write(buf, binary.BigEndian, header.qdcount); err != nil {
		return n, fmt.Errorf("Failed to write qdcount: %s", err)
	}
	n += 2
	if err = binary.Write(buf, binary.BigEndian, header.ancount); err != nil {
		return n, fmt.Errorf("Failed to write ancount: %s", err)
	}
	n += 2
	if err = binary.Write(buf, binary.BigEndian, header.nscount); err != nil {
		return n, fmt.Errorf("Failed to write nscount: %s", err)
	}
	n += 2
	if err = binary.Write(buf, binary.BigEndian, header.artcount); err != nil {
		return n, fmt.Errorf("Failed to write artcount: %s", err)
	}
	n += 2
	return n, nil
}

//******************************************************************************************** Name
// TODO: maybe break this up? kinda ugly :(
func deserialName(r *bytes.Reader, namemap map[uint16]string, offset uint16) (name string, read uint16, err error) {
	ptrUsed := false
	segN := 0
	for {
		nbuf := make([]byte, 1)
		n, err := io.ReadAtLeast(r, nbuf, 1)
		if err != nil || n != 1 {
			return name, read, fmt.Errorf("Failed to read first byte of name: %s\n", err)
		}
		read += uint16(n)
		firstByte := uint8(nbuf[0])
		if firstByte == 0 {
			// End of domain
			break
		} else if (firstByte & 0b1100_0000) == 0b1100_0000 {
			ptrUsed = true
			// Compressed name
			halfOffset := uint16(firstByte&0b0011_1111) << 8
			n, err := io.ReadAtLeast(r, nbuf, 1)
			if err != nil || n != 1 {
				return name, read, fmt.Errorf("Failed to read first byte of name: %s\n", err)
			}
			read += uint16(n)
			byteOffset := halfOffset | uint16(nbuf[0])
			if segment, ok := namemap[byteOffset]; !ok {
				return name, read, fmt.Errorf("byteOffset %X not in map!", byteOffset)
			} else {
				name += segment
			}
		} else if firstByte >= 63 {
			// Something has gone terribly wrong
			return name, read, fmt.Errorf("Unexpected segment length (>= 63)")
		}
		// Otherwise it's the length of the next segment
		seglen := int(firstByte)
		buf := make([]byte, seglen)
		n, err = io.ReadAtLeast(r, buf, seglen)
		if err != nil || n != seglen {
			return name, read, fmt.Errorf("Expected %d bytes, got %d bytes. Error: %s\n", seglen, n, err)
		}
		read += uint16(n)
		if segN != 0 {
			name += "." + string(buf)
		} else {
			name += string(buf)
		}
		segN += 1
	}

	if !ptrUsed {
		namemap[offset] = name
	}

	return name, read, nil
}

func writePointer(buf *bytes.Buffer, ptr uint16) (offset uint16, err error) {
	if (ptr & 0b1100_0000_0000_0000) != 0 {
		return offset, fmt.Errorf("Pointer %X location too large!", ptr)
	}
	ptr |= 0b1100_0000_0000_0000

	if err = binary.Write(buf, binary.BigEndian, ptr); err != nil {
		return offset, fmt.Errorf("Failed to write seglength: %s", err)
	}
	return 2, nil
}

// TODO: add compression here (too lazy rn)
func serialName(buf *bytes.Buffer, name string) (offset uint16, err error) {
	// https://en.wikipedia.org/wiki/Hostname#Restrictions_on_valid_host_names
	n := len(name)
	if n > 253 {
		return offset, fmt.Errorf("Total domain name length %d is larger than the max 253\n", n)
	}

	for _, segment := range strings.Split(name, ".") {
		seglen := uint8(len(segment))
		if seglen > 62 {
			return offset, fmt.Errorf("Domain has a segment longer than 62 characters")
		}
		if err = binary.Write(buf, binary.BigEndian, seglen); err != nil {
			return offset, fmt.Errorf("Failed to write seglength: %s", err)
		}
		for _, c := range segment {
			if err = binary.Write(buf, binary.BigEndian, byte(c)); err != nil {
				return offset, fmt.Errorf("Failed to write character: %s", err)
			}
		}
	}

	if err = binary.Write(buf, binary.BigEndian, uint8(0)); err != nil {
		return offset, fmt.Errorf("Failed to write null terminator: %s", err)
	}
	return offset, nil
}

func contains(namemap map[uint16]string, domain string) (k uint16, ok bool) {
	for k, v := range namemap {
		if v == domain {
			return k, true
		}
	}
	return k, false
}

//**************************************************************************************** Question
func serialQuestion(buf *bytes.Buffer, namemap map[uint16]string, offset uint16, question *Question) (written uint16, err error) {

	if k, ok := contains(namemap, question.qname); ok {
		n, err := writePointer(buf, k)
		if err != nil {
			return written, fmt.Errorf("Failed to write pointer: %s", err)
		}
		written += n
	} else {
		namemap[offset] = question.qname
		n, err := serialName(buf, question.qname)
		written += n
		if err != nil {
			return written, fmt.Errorf("Failed to serialize response name: %s", err)
		}
	}
	if err = binary.Write(buf, binary.BigEndian, question.qtype); err != nil {
		return written, fmt.Errorf("Failed to write qtype: %s", err)
	}
	written += 2

	if err = binary.Write(buf, binary.BigEndian, question.qclass); err != nil {
		return written, fmt.Errorf("Failed to write qclass: %s", err)
	}
	written += 2
	return written, nil
}

func deserialQuestion(r *bytes.Reader, namemap map[uint16]string, offset uint16, question *Question) (read uint16, err error) {
	name, n, err := deserialName(r, namemap, offset)
	if err != nil {
		return read, fmt.Errorf("Failed to deserialize request name: %s", err)
	}
	read += uint16(n)
	question.qname = name

	if err = binary.Read(r, binary.BigEndian, &question.qtype); err != nil {
		return read, fmt.Errorf("Failed to read qtype: %s", err)
	}
	read += 2

	if err = binary.Read(r, binary.BigEndian, &question.qclass); err != nil {
		return read, fmt.Errorf("Failed to read qclass: %s", err)
	}
	read += 2

	return read, nil
}

//****************************************************************************************** Answer
func deserialAnswer(r *bytes.Reader, namemap map[uint16]string, offset uint16, answer *Answer) (read uint16, err error) {
	name, n, err := deserialName(r, namemap, offset)
	if err != nil {
		return read, fmt.Errorf("Failed to deserialize request name: %s", err)
	}
	read += n
	answer.name = name

	if err = binary.Read(r, binary.BigEndian, answer.kind); err != nil {
		return read, fmt.Errorf("Failed to read kind(aka type): %s", err)
	}
	read += 2
	if err = binary.Read(r, binary.BigEndian, answer.class); err != nil {
		return read, fmt.Errorf("Failed to read class: %s", err)
	}
	read += 2
	if err = binary.Read(r, binary.BigEndian, answer.ttl); err != nil {
		return read, fmt.Errorf("Failed to read ttl: %s", err)
	}
	read += 4
	if err = binary.Read(r, binary.BigEndian, answer.rdlength); err != nil {
		return read, fmt.Errorf("Failed to read rdlength: %s", err)
	}
	read += 2
	return read, nil
}

func serialAnswer(buf *bytes.Buffer, namemap map[uint16]string, offset uint16, answer *Answer) (written uint16, err error) {
	if k, ok := contains(namemap, answer.name); ok {
		n, err := writePointer(buf, k)
		if err != nil {
			return written, fmt.Errorf("Failed to write pointer: %s", err)
		}
		written += n
	} else {
		namemap[offset] = answer.name
		n, err := serialName(buf, answer.name)
		written += n
		if err != nil {
			return written, fmt.Errorf("Failed to serialize response name: %s", err)
		}
	}

	if err = binary.Write(buf, binary.BigEndian, answer.kind); err != nil {
		return written, fmt.Errorf("Failed to write kind(aka type): %s", err)
	}
	written += 2

	if err = binary.Write(buf, binary.BigEndian, answer.class); err != nil {
		return written, fmt.Errorf("Failed to write class: %s", err)
	}
	written += 2

	if err = binary.Write(buf, binary.BigEndian, answer.ttl); err != nil {
		return written, fmt.Errorf("Failed to write ttl: %s", err)
	}
	written += 4

	if err = binary.Write(buf, binary.BigEndian, answer.rdlength); err != nil {
		return written, fmt.Errorf("Failed to write rdlength: %s", err)
	}
	written += 2

	// TODO: maybe this section is unncessary? Maybe this processing shouldn't happen here?
	switch answer.kind {
	case 0x0001:
		if err = binary.Write(buf, binary.BigEndian, answer.rdata); err != nil {
			return written, fmt.Errorf("Failed to write rdata: %s", err)
		}
		written += 4
	case 0x0002:
	case 0x0005:
	case 0x000F:
	default:
		// These other "types" should be unsupported for now
		// We probably(?) don't need them
		return written, fmt.Errorf("response.kind is invalid: %X\n", answer.kind)
	}
	return written, nil
}

//******************************************************************************************* Auth.
// TODO: these are no-ops for now. I believe we don't need them yet. But, whatever impl. we do,
//       according to the RFC, they should be the same as Answer.
func serialAuthority(buf *bytes.Buffer, authority *Authority) (err error) {
	return nil
}

func deserialAuthority(r *bytes.Reader, authority *Authority) (err error) {
	return nil
}

//******************************************************************************************* Addl.
// TODO: these are no-ops for now. I believe we don't need them yet. But, whatever impl. we do,
//       according to the RFC, they should be the same as Answer.
func serialAdditional(buf *bytes.Buffer, additional *Additional) (err error) {
	return nil
}
func deserialAdditional(r *bytes.Reader, additional *Additional) (err error) {
	return nil
}
