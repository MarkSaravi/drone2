package icm20948

import (
	"time"

	"github.com/MarkSaravi/drone-go/types"
	"github.com/MarkSaravi/drone-go/utils"
	"periph.io/x/periph/conn/physic"
	"periph.io/x/periph/conn/spi"
	"periph.io/x/periph/host"
	"periph.io/x/periph/host/sysfs"
)

func reg(reg uint16) *Register {
	return &Register{
		address: byte(reg),
		bank:    byte(reg >> 8),
	}
}

var accelerometerSensitivity = make(map[int]float64)
var gyroFullScale = make(map[int]float64)

func init() {
	accelerometerSensitivity[0] = SENSITIVITY_0
	accelerometerSensitivity[1] = SENSITIVITY_1
	accelerometerSensitivity[2] = SENSITIVITY_2
	accelerometerSensitivity[3] = SENSITIVITY_3

	gyroFullScale[0] = SCALE_0
	gyroFullScale[1] = SCALE_1
	gyroFullScale[2] = SCALE_2
	gyroFullScale[3] = SCALE_3

	host.Init()
}

// NewICM20948Driver creates ICM20948 driver for raspberry pi
func NewICM20948Driver(
	busNumber int,
	chipSelect int,
	config DeviceConfig,
	accConfig AccelerometerConfig,
	gyroConfig GyroscopeConfig,

) (*Device, error) {
	d, err := sysfs.NewSPI(busNumber, chipSelect)
	if err != nil {
		return nil, err
	}
	conn, err := d.Connect(7*physic.MegaHertz, spi.Mode3, 8)
	if err != nil {
		return nil, err
	}
	dev := Device{
		SPI:     d,
		Conn:    conn,
		regbank: 0xFF,
		acc: threeAxis{
			data:     types.XYZ{X: 0, Y: 0, Z: 0},
			prevData: types.XYZ{X: 0, Y: 0, Z: 0},
			dataDiff: 0,
			config:   accConfig,
		},
		gyro: threeAxis{
			data:     types.XYZ{X: 0, Y: 0, Z: 0},
			prevData: types.XYZ{X: 0, Y: 0, Z: 0},
			dataDiff: 0,
			config:   gyroConfig,
		},
	}
	return &dev, nil
}

func (dev *Device) readReg(address byte, len int) ([]byte, error) {
	w := make([]byte, len+1)
	r := make([]byte, len+1)
	w[0] = (address & 0x7F) | 0x80
	err := dev.Conn.Tx(w, r)
	return r[1:], err
}

func (dev *Device) writeReg(address byte, data ...byte) error {
	if len(data) == 0 {
		return nil
	}
	w := append([]byte{address & 0x7F}, data...)
	err := dev.Conn.Tx(w, nil)
	return err
}

func (dev *Device) selRegisterBank(regbank byte) error {
	if regbank == dev.regbank {
		return nil
	}
	dev.regbank = regbank
	return dev.writeReg(REG_BANK_SEL, (regbank<<4)&0x30)
}

func (dev *Device) readRegister(register uint16, len int) ([]byte, error) {
	reg := reg(register)
	dev.selRegisterBank(reg.bank)
	return dev.readReg(reg.address, len)
}

func (dev *Device) writeRegister(register uint16, data ...byte) error {
	if len(data) == 0 {
		return nil
	}
	reg := reg(register)
	dev.selRegisterBank(reg.bank)
	return dev.writeReg(reg.address, data...)
}

// WhoAmI return value for ICM-20948 is 0xEA
func (dev *Device) WhoAmI() (name string, id byte, err error) {
	name = "ICM-20948"
	data, err := dev.readRegister(WHO_AM_I, 1)
	id = data[0]
	return
}

// GetDeviceConfig reads device configurations
func (dev *Device) GetDeviceConfig() (
	config types.Config,
	accConfig types.Config,
	gyroConfig types.Config,
	err error) {
	// data, err := dev.readRegister(LP_CONFIG, 3)
	config = DeviceConfig{}
	accConfig, err = dev.getAccConfig()
	gyroConfig, err = dev.getGyroConfig()
	return
}

// InitDevice applies initial configurations to device
func (dev *Device) InitDevice() error {
	// Reset settings to default
	err := dev.writeRegister(PWR_MGMT_1, 0b10000000)
	time.Sleep(50 * time.Millisecond) // wait for taking effect
	data, err := dev.readRegister(PWR_MGMT_1, 1)
	const nosleep byte = 0b10111111
	config := byte(data[0] & nosleep)
	const accGyro byte = 0b00000000
	err = dev.writeRegister(PWR_MGMT_1, config, accGyro)
	time.Sleep(50 * time.Millisecond) // wait for taking effect
	err = dev.InitAccelerometer()
	time.Sleep(50 * time.Millisecond) // wait for taking effect
	err = dev.InitGyroscope()
	time.Sleep(50 * time.Millisecond) // wait for taking effect
	return err
}

// ReadRawData reads all Accl and Gyro data
func (dev *Device) ReadRawData() ([]byte, error) {
	return dev.readRegister(ACCEL_XOUT_H, 12)
}

// Start starts device
func (dev *Device) Start() {
	dev.lastReading = time.Now().UnixNano()
}

// ReadData reads Accelerometer and Gyro data
func (dev *Device) ReadData() (acc types.XYZ, gyro types.XYZ, err error) {
	data, err := dev.ReadRawData()
	now := time.Now().UnixNano()
	dev.duration = dev.lastReading - now
	dev.lastReading = now
	dev.processAccelerometerData(data)
	dev.processGyroscopeData(data[6:])
	return dev.GetAcc().GetData(), dev.GetGyro().GetData(), err
}

func (a *threeAxis) GetConfig() types.Config {
	return a.config
}

func (a *threeAxis) SetConfig(config types.Config) {
	a.config = config
}

func (a *threeAxis) GetData() types.XYZ {
	return a.data
}

func (a *threeAxis) SetData(x, y, z float64) {
	a.prevData = a.data
	a.data = types.XYZ{
		X: x,
		Y: y,
		Z: z,
	}
	a.dataDiff = utils.CalcVectorLen(a.data) - utils.CalcVectorLen(a.prevData)
}

func (a *threeAxis) GetDiff() float64 {
	return a.dataDiff
}
