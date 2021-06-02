package flightcontrol

import (
	"fmt"
	"math"

	"github.com/MarkSaravi/drone-go/modules/imu"
	"github.com/MarkSaravi/drone-go/types"
	"github.com/MarkSaravi/drone-go/utils"
)

type FlightStates struct {
	Config         types.FlightConfig
	ImuDataChannel <-chan imu.ImuData
	imuData        imu.ImuData
	accRotations   types.Rotations
	gyroRotations  types.Rotations
	rotations      types.Rotations
}

func (fs *FlightStates) Reset() {
	fs.gyroRotations = types.Rotations{
		Roll:  0,
		Pitch: 0,
		Yaw:   0,
	}
}

func (fs *FlightStates) Set(imuData imu.ImuData) {
	fs.imuData = imuData
	fs.setAccRotations(fs.Config.AccLowPassFilterCoefficient)
	fs.setGyroRotations()
	fs.setRotations()
}

func goDurToDt(d int64) float64 {
	return float64(d) / 1e9
}

func getOffset(offset float64, dt float64) float64 {
	return dt * offset
}

func accelerometerDataToRollPitch(data types.XYZ) (roll, pitch float64) {
	roll = utils.RadToDeg(math.Atan2(data.Y, data.Z))
	pitch = -utils.RadToDeg(math.Atan2(data.X, data.Z))
	return
}

func gyroscopeDataToRollPitchYawChange(wg types.XYZ, readingInterval int64) (
	float64, float64, float64) { // angular velocity
	dt := goDurToDt(readingInterval) // reading interval
	return wg.X * dt, wg.Y * dt, wg.X * dt
}

func (fs *FlightStates) setAccRotations(lowPassFilterCoefficient float64) {
	roll, pitch := accelerometerDataToRollPitch(fs.imuData.Acc.Data)
	fs.accRotations = types.Rotations{
		Roll:  roll,
		Pitch: pitch,
		Yaw:   0,
	}
}

func (fs *FlightStates) setGyroRotations() {
	curr := fs.gyroRotations // current rotations by gyro
	dRoll, dPitch, dYaw := gyroscopeDataToRollPitchYawChange(
		fs.imuData.Gyro.Data,
		fs.imuData.ReadInterval,
	)
	fs.gyroRotations = types.Rotations{
		Roll:  curr.Roll + dRoll,
		Pitch: curr.Pitch + dPitch,
		Yaw:   curr.Yaw + dYaw,
	}
}

func (fs *FlightStates) setRotations() {
	fs.rotations = types.Rotations{
		Roll:  fs.accRotations.Roll,
		Pitch: fs.accRotations.Pitch,
		Yaw:   fs.accRotations.Yaw,
	}
}

func (fs *FlightStates) ImuDataToJson() string {
	return fmt.Sprintf("{\"accRoll\":%0.2f,\"accPitch\":%0.2f,\"accYaw\":%0.2f,\"gyroRoll\":%0.2f,\"gyroPitch\":%0.2f,\"gyroYaw\":%0.2f,\"roll\":%0.2f,\"ritch\":%0.2f,\"yaw\":%0.2f,\"dT\":%d,\"T\":%d}",
		fs.accRotations.Roll,
		fs.accRotations.Pitch,
		fs.accRotations.Yaw,
		fs.gyroRotations.Roll,
		fs.gyroRotations.Pitch,
		fs.gyroRotations.Yaw,
		fs.rotations.Roll,
		fs.rotations.Pitch,
		fs.rotations.Yaw,
		fs.imuData.ReadInterval,
		fs.imuData.ReadTime,
	)
}
