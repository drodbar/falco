package main

import (
	"os"

	"github.com/pkg/errors"
	"github.com/ysugimoto/falco/tester/shared"
)

func writeLCOVFile(factory *shared.CoverageFactory, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()
	return errors.WithStack(factory.WriteLCOV(f, ""))
}
