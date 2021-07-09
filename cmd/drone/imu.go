package main

import (
	"os"

	"github.com/MarkSaravi/drone-go/devices/icm20948"
	"github.com/MarkSaravi/drone-go/modules/imu"
	"github.com/MarkSaravi/drone-go/types"
)

func initiateIMU(config ApplicationConfig) types.IMU {
	dev, err := icm20948.NewICM20948Driver(config.Devices.ICM20948)
	if err != nil {
		os.Exit(1)
	}
	dev.InitDevice()
	if err != nil {
		os.Exit(1)
	}
	imudevice := imu.NewIMU(dev, config.Flight.Imu)
	return &imudevice
}

func createImuDataChannel(config ApplicationConfig) chan types.ImuRotations {
	imuDataChannel := make(chan types.ImuRotations, 1)
	imu := initiateIMU(config)
	imu.ResetReadingTimes()
	go func() {
		for {
			rotations, err := imu.GetRotations()
			if err == nil {
				imuDataChannel <- rotations
			}
		}
	}()
	return imuDataChannel
}
