package main

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func byteSlice(data string) (out []byte) {
	out, err := hex.DecodeString(strings.ReplaceAll(data, " ", ""))
	if err != nil {
		panic(err)
	}
	return out
}

func prettyHex(data []byte) {
	for i, b := range data {
		if i%32 == 0 {
			fmt.Printf("\n")
		}
		fmt.Printf("%02X ", b)
	}
	fmt.Print("\n\n")
}

func TestSerialize(t *testing.T) {
	cases := []struct {
		res     Message
		want    []byte
		wantErr error
	}{
		{
			res: Message{
				header: Header{
					id:      0xdd06,
					packed:  packed(1, 0, 0, 0, 1, 1, 0, 0),
					qdcount: 1,
					ancount: 1,
				},
				questions: []Question{
					{qname: "google.com", qtype: 1, qclass: 1},
				},
				answers: []Answer{
					// TODO: UGH name's gotta be compressed
					// 142.251.214.142
					// 8E.FB.D6.8E
					{name: "google.com", kind: 0x0001, class: 1, ttl: 47, rdlength: 4, rdata: []byte{uint8(142), uint8(251), uint8(214), uint8(142)}},
				},
			},
			want: byteSlice("dd 06 81 80 00 01 00 01 00 00 00 00 06 67 6f 6f" +
				"67 6c 65 03 63 6f 6d 00 00 01 00 01 c0 0c 00 01" +
				"00 01 00 00 00 2f 00 04 8e fb d6 8e"),
			wantErr: nil},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			got, gotErr := Serialize(tc.res)
			if gotErr != tc.wantErr {
				t.Errorf("Got error %s, wanted %s\n", gotErr, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				fmt.Print("Got:")
				prettyHex(got)
				fmt.Print("Want:")
				prettyHex(tc.want)
				t.Errorf("Packets not equal")
			}
		})
	}
}

func TestDeserialize(t *testing.T) {
	cases := []struct {
		req     []byte
		want    Message
		wantErr error
	}{
		{
			want: Message{
				header: Header{
					id:      0xdd06,
					packed:  packed(0, 0, 0, 0, 1, 2, 0, 0),
					qdcount: 1,
					ancount: 1,
				},
				questions: []Question{
					{qname: "google.com", qtype: 1, qclass: 1},
				},
				answers: []Answer{
					// TODO: UGH name's gotta be compressed
					// 142.251.214.142
					// 8E.FB.D6.8E
					{name: "google.com", kind: 0x0001, class: 1, ttl: 47, rdlength: 4, rdata: []byte{uint8(142), uint8(251), uint8(214), uint8(142)}},
				},
			},
			req: byteSlice("dd 06 01 20 00 01 00 00 00 00 00 00 06 67 6f 6f" +
				"67 6c 65 03 63 6f 6d 00 00 01 00 01"),
			wantErr: nil},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			got, gotErr := Deserialize(tc.req)
			if gotErr != tc.wantErr {
				t.Errorf("Got error %s, wanted %s\n", gotErr, tc.wantErr)
			}
			if reflect.DeepEqual(got, tc.want) {
				t.Errorf("Got error %X, wanted %X\n", got, tc.want)
			}
		})
	}
}

func TestPacking(t *testing.T) {
	for a := uint16(0); a < uint16(1<<16-1); a += 1 {
		b := packed(unpacked(a))
		if a != b {
			t.Errorf("Failed to pack(unpack(0x%04x))\n Got: 0x%04x\n", a, b)
			break
		}
	}
}
