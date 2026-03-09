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

func writeCoverageFile(factory *shared.CoverageFactory, path, format string) error {
	f, err := os.Create(path)
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()

	switch format {
	case "generic-xml":
		return errors.WithStack(factory.WriteGenericXML(f, ""))
	default:
		return errors.WithStack(factory.WriteLCOV(f, ""))
	}
}
