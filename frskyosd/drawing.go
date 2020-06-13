package frskyosd

import (
	"encoding/binary"
	"fmt"
)

type Color uint

const (
	CBlack Color = iota
	CTransparent
	CWhite
	CGray
)

func (o *OSD) pack2int12(x int, y int) ([]byte, error) {
	const (
		min = -(1 << 12)
		max = (1 << 12) - 1
	)
	if x < min || x > max {
		return nil, fmt.Errorf("x = %v is out of bounds %d-%d", x, min, max)
	}
	if y < min || y > max {
		return nil, fmt.Errorf("y = %v is out of bounds %d-%d", y, min, max)
	}
	buf := make([]byte, 4)
	x16 := uint32(int16(x))
	y16 := uint32(int16(y))
	binary.LittleEndian.PutUint32(buf, y16<<12|x16)
	return buf[:3], nil
}

func (o *OSD) TransactionBegin() error {
	return o.send(cmdTransactionBegin, nil)
}

func (o *OSD) TransactionCommit() error {
	return o.send(cmdTransactionCommit, nil)
}

func (o *OSD) TransactionBeginResettingDrawing() error {
	return o.send(cmdTransactionBeginResetDrawing, nil)
}

func (o *OSD) checkColor(c Color) error {
	if c > CGray {
		return fmt.Errorf("invalid color %d", uint(c))
	}
	return nil
}

func (o *OSD) setColor(cmd osdCmd, c Color) error {
	if err := o.checkColor(c); err != nil {
		return err
	}
	return o.send(cmd, []byte{byte(c)})
}

func (o *OSD) SetStrokeColor(c Color) error {
	return o.setColor(cmdSetStrokeColor, c)
}

func (o *OSD) SetFillColor(c Color) error {
	return o.setColor(cmdSetFillColor, c)
}

func (o *OSD) SetStrokeWidth(w int) error {
	return o.send(cmdSetStrokeColor, []byte{byte(w)})
}

func (o *OSD) ClearScreen() error {
	return o.send(cmdClearScreen, nil)
}

func (o *OSD) ResetDrawing() error {
	return o.send(cmdDrawingReset, nil)
}

func (o *OSD) MoveToPoint(x int, y int) error {
	data, err := o.pack2int12(x, y)
	if err != nil {
		return err
	}
	return o.send(cmdMoveToPoint, data)
}

func (o *OSD) StrokeLineToPoint(x int, y int) error {
	data, err := o.pack2int12(x, y)
	if err != nil {
		return err
	}
	return o.send(cmdStrokeLineToPoint, data)
}

func (o *OSD) FillRect(x int, y int, w uint, h uint) error {
	origin, err := o.pack2int12(x, y)
	if err != nil {
		return err
	}
	size, err := o.pack2int12(int(w), int(h))
	if err != nil {
		return err
	}
	data := make([]byte, 0, len(origin)+len(size))
	data = append(data, origin...)
	data = append(data, size...)
	return o.send(cmdFillRect, data)
}
