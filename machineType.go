package infra

import (
	"errors"
	"fmt"
)

type StandardType string
type MachineType struct {
	// CPUCount should be either 1 or an even number upto 32.
	CPUCount int `json:"cpu_count"`

	// MemoryMBs must be a multiple of 256MB.
	MemoryMBs int `json:"memory_mbs"`

	Type StandardType `json:"type"`
}

var (
	errInvalidZeroCount    = errors.New("expecting 1 or an even CPU count upto 32")
	errMemoryMultipleOf256 = errors.New("memory must be a multiple of 256")
	errEmptyType           = errors.New("expecting a non-empty type")
)

func (mt *MachineType) Validate() error {
	validators := []func() error{
		mt.validateAsCustomMachine,
		mt.validateAsStandardMachine,
	}

	var err error
	for _, fn := range validators {
		err = fn()
		if err == nil {
			return nil
		}
	}
	return err
}

func (mt *MachineType) validateAsStandardMachine() error {
	if mt.Type == "" {
		return errEmptyType
	}
	return nil
}

func (mt *MachineType) validateAsCustomMachine() error {
	if mt.CPUCount <= 0 || mt.CPUCount > 32 {
		return errInvalidZeroCount
	}

	// Only either 1 or an even number upto 32.
	if !(mt.CPUCount == 1 || (1&mt.CPUCount) == 0) {
		return errInvalidZeroCount
	}

	// Now memory has to be a multiple of 256
	if ((mt.MemoryMBs / 256) * 256) != mt.MemoryMBs {
		return errMemoryMultipleOf256
	}

	return nil
}

func (mt *MachineType) standardRoute() string {
	return fmt.Sprintf("/machineTypes/%s", mt.Type)
}

func (mt *MachineType) customRoute() string {
	// /machineTypes/custom-CPUS-MEMORY
	return fmt.Sprintf("/machineTypes/custom-%d-%d", mt.CPUCount, mt.MemoryMBs)
}

func (mt *MachineType) partialURLByZone(zone string) string {
	return fmt.Sprintf("/zones/%s%s", zone, mt.route())
}

func (mt *MachineType) canMakeCustomMachine() bool {
	return mt.validateAsCustomMachine() == nil
}

func (mt *MachineType) canMakeStandardMachine() bool {
	return mt.validateAsStandardMachine() == nil
}

func (mt *MachineType) route() string {
	if mt.canMakeCustomMachine() {
		return mt.customRoute()
	}
	return mt.standardRoute()
}

// Predefined machine types
var (
	basic1VCPUMachine = &MachineType{
		Type: N1Standard1,
	}
)

const (
	// Standard types defined at https://cloud.google.com/compute/docs/machine-types
	// As of Mon 29 May 2017 16:59:31 PDT.

	// Standard machine type with 1 virtual CPU and 3.75 GB of memory.
	// Maximum number of persistent disks: 16 (32 in Beta).
	// Maximum total of persistent disk size: 64 TB.
	N1Standard1 StandardType = "n1-standard-1"

	// Standard machine type with 2 virtual CPU and 7.50 GB of memory.
	// Maximum number of persistent disks: 16 (64 in Beta).
	// Maximum total of persistent disk size: 64 TB.
	N1Standard2 StandardType = "n1-standard-2"

	// Standard machine type with 4 virtual CPU and 15 GB of memory.
	// Maximum number of persistent disks: 16 (64 in Beta).
	// Maximum total of persistent disk size: 64 TB.
	N1Standard4 StandardType = "n1-standard-4"

	// Standard machine type with 8 virtual CPU and 30 GB of memory.
	// Maximum number of persistent disks: 16 (64 in Beta).
	// Maximum total of persistent disk size: 64 TB.
	N1Standard8 StandardType = "n1-standard-8"

	// Standard machine type with 16 virtual CPU and 60 GB of memory.
	// Maximum number of persistent disks: 16 (128 in Beta).
	// Maximum total of persistent disk size: 64 TB.
	N1Standard16 StandardType = "n1-standard-16"

	// Standard machine type with 32 virtual CPU and 120 GB of memory.
	// Maximum number of persistent disks: 16 (128 in Beta).
	// Maximum total of persistent disk size: 64 TB.
	N1Standard32 StandardType = "n1-standard-32"

	// Standard machine type with 64 virtual CPU and 240 GB of memory.
	// Maximum number of persistent disks: 16 (128 in Beta).
	// Maximum total of persistent disk size: 64 TB.
	N1Standard64 StandardType = "n1-standard-64"
)
