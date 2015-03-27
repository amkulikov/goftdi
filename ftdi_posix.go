package ftdi

import "errors"
import "unsafe"

// If installed on OSX using 'brew'
// ;;#cgo CFLAGS: -I/usr/local/Cellar/libftdi/1.1/include/libftdi1/
// ;;#cgo LDFLAGS: -lftdi1 -L/usr/local/Cellar/libftdi/1.1/lib/

// #cgo pkg-config: libftdi
// #include <ftdi.h>
import "C"

// Return Library version, formatted to match D2XX
func GetLibraryVersion() uint32 {
	//v := C.ftdi_get_library_version()
	//return uint32(v.major&0xFF<<16 +
	//	v.minor&0xFF<<8 +
	//	v.micro&0xFF)

	// Not implemented, so we assume version 0.19.0
	return 0x001900
}

type DeviceInfo struct {
	index         uint64
	id            uint32 // used as interface number
	serial_number string
	description   string
	manufacturer  string
	handle        unsafe.Pointer // the libusb device pointer
}

//TODO: Need to expand multi-interface devices, and then to other FTDI chips
func GetDeviceList() (dl []DeviceInfo, e error) {
	ctx := C.ftdi_new()
	defer C.ftdi_free(ctx)
	if ctx == nil {
		return nil, errors.New("Failed to create FTDI context")
	}

	var dev_list *C.struct_ftdi_device_list
	defer C.ftdi_list_free(&dev_list)

	num := C.ftdi_usb_find_all(ctx, &dev_list, 0x0403, 0x6011)
	if num < 0 {
		return nil, getErr(ctx)
	}

	dl = make([]DeviceInfo, num*4)

	for i := 0; i < int(num); i++ {

		const CHAR_SZ = 64
		var mnf_char, desc_char, ser_char [CHAR_SZ]C.char

		ret := C.ftdi_usb_get_strings(ctx, dev_list.dev,
			(*C.char)(&mnf_char[0]), CHAR_SZ,
			(*C.char)(&desc_char[0]), CHAR_SZ,
			(*C.char)(&ser_char[0]), CHAR_SZ)
		if ret != 0 {
			return nil, getErr(ctx)
		}

		var d DeviceInfo
		d.handle = unsafe.Pointer(dev_list.dev)
		d.manufacturer = C.GoString(&mnf_char[0])

		for j, intrfce := range []string{"A", "B", "C", "D"} {
			d.index = uint64(i*4 + j)
			d.id = uint32(j)
			d.description = C.GoString(&desc_char[0]) + " " + intrfce
			d.serial_number = C.GoString(&ser_char[0]) + intrfce
			dl[d.index] = d
		}
		dev_list = dev_list.next
	}

	return dl, nil
}

type Device struct {
	ctx *C.struct_ftdi_context
}

func Open(di DeviceInfo) (d *Device, e error) {
	ctx := C.ftdi_new()
	if ctx == nil {
		C.ftdi_free(ctx)
		return d, errors.New("Failed to create FTDI context")
	}

	if ret := C.ftdi_usb_open_dev(ctx, (*C.struct_usb_device)(di.handle)); ret != 0 {
		C.ftdi_free(ctx)
		return d, getErr(ctx)
	}

	if ret := C.ftdi_set_interface(ctx, di.id); ret != 0 {
		C.ftdi_free(ctx)
		return d, getErr(ctx)
	}

	return &Device{ctx}, nil
}

func (d *Device) Close() (e error) {
	defer C.ftdi_free(d.ctx)
	if ret := C.ftdi_usb_close(d.ctx); ret != 0 {
		return getErr(d.ctx)
	}
	return nil
}

func (d *Device) GetStatus() (rx_queue, tx_queue, events int32, e error) {
	return 0, 0, 0, errors.New("Not Implemented")
}

func (d *Device) Read(p []byte) (n int, e error) {
	ret := C.ftdi_read_data(d.ctx, (*C.uchar)(&p[0]), C.int(len(p)))
	if ret < 0 {
		return 0, getErr(d.ctx)
	}
	return int(ret), nil
}

func (d *Device) Write(p []byte) (n int, e error) {
	ret := C.ftdi_write_data(d.ctx, (*C.uchar)(&p[0]), C.int(len(p)))
	if ret < 0 {
		return 0, getErr(d.ctx)
	}
	return int(ret), nil
}

func (d *Device) SetBaudRate(baud uint) (e error) {
	if ret := C.ftdi_set_baudrate(d.ctx, C.int(baud)); ret < 0 {
		return getErr(d.ctx)
	}
	return nil
}

func (d *Device) SetChars(event, err byte) (e error) {
	if ret := C.ftdi_set_event_char(d.ctx, C.uchar(event), C.uchar(event)); ret < 0 {
		return getErr(d.ctx)
	}
	if ret := C.ftdi_set_error_char(d.ctx, C.uchar(err), C.uchar(err)); ret < 0 {
		return getErr(d.ctx)
	}
	return nil
}

func (d *Device) SetBitMode(mode BitMode) (e error) {
	const mask = 0x00
	if ret := C.ftdi_set_bitmode(d.ctx, mask, C.uchar(mode)); ret < 0 {
		return getErr(d.ctx)
	}
	return nil
}

func (d *Device) SetFlowControl(f FlowControl) (e error) {
	if ret := C.ftdi_setflowctrl(d.ctx, C.int(f)); ret < 0 {
		return getErr(d.ctx)
	}
	return nil
}

func (d *Device) SetLatency(latency int) (e error) {
	if ret := C.ftdi_set_latency_timer(d.ctx, C.uchar(latency)); ret < 0 {
		return getErr(d.ctx)
	}
	return nil
}

func (d *Device) SetTransferSize(read_size, write_size int) (e error) {
	if ret := C.ftdi_read_data_set_chunksize(d.ctx, C.uint(read_size)); ret < 0 {
		return getErr(d.ctx)
	}
	if ret := C.ftdi_write_data_set_chunksize(d.ctx, C.uint(write_size)); ret < 0 {
		return getErr(d.ctx)
	}
	return nil
}

func (d *Device) SetLineProperty(props LineProperties) (e error) {
	if ret := C.ftdi_set_line_property(d.ctx,
		uint32(props.Bits),
		uint32(props.StopBits),
		uint32(props.Parity)); ret < 0 {
		return getErr(d.ctx)
	}
	return nil
}

func (d *Device) SetTimeout(read_timeout, write_timeout int) (e error) {
	// NOP
	return nil
}

func (d *Device) Reset() (e error) {
	if ret := C.ftdi_usb_reset(d.ctx); ret < 0 {
		return getErr(d.ctx)
	}
	return nil
}

func (d *Device) Purge() (e error) {
	if ret := C.ftdi_usb_purge_buffers(d.ctx); ret < 0 {
		return getErr(d.ctx)
	}
	return nil
}

func getErr(ctx *C.struct_ftdi_context) error {
	return errors.New(C.GoString(C.ftdi_get_error_string(ctx)))
}
