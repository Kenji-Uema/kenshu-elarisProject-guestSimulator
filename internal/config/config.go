package config

type BookingMachineConfig struct {
	ClockEmuUrl       string
	CottageManagerUrl string
	GuestManagerUrl   string
	GraphFile         string
}

type GuestRegisterMachineConfig struct {
	GuestManagerUrl string
	GraphFile       string
}
