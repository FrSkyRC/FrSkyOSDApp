package frskyosd

import (
	"github.com/go-daq/crc8"
)

var (
	crc8D5Table = crc8.MakeTable(0xD5)

	_ checkSum = (*crc8D5Checksum)(nil)
)

type checkSum interface {
	WriteByte(b byte) error
	Sum8() uint8
}

type crc8D5Checksum struct {
	crc crc8.Hash8
}

func (c *crc8D5Checksum) WriteByte(b byte) error {
	_, err := c.crc.Write([]byte{b})
	return err
}

func (c *crc8D5Checksum) Sum8() uint8 {
	return c.crc.Sum8()
}

func newCrc8D5Checksum() checkSum {
	return &crc8D5Checksum{
		crc: crc8.New(crc8D5Table),
	}
}

type xorChecksum struct {
	sum uint8
}

func (c *xorChecksum) WriteByte(b byte) error {
	c.sum ^= b
	return nil
}

func (c *xorChecksum) Sum8() uint8 {
	return c.sum
}

func newXorChecksum() checkSum {
	return &xorChecksum{}
}

func checkSumWrite(cs checkSum, data []byte) error {
	for _, b := range data {
		if err := cs.WriteByte(b); err != nil {
			return err
		}
	}
	return nil
}
