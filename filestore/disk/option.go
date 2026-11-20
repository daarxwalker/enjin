package disk

import "io/fs"

type Option func(*Driver)

func WithFileMode(mode fs.FileMode) Option {
	return func(d *Driver) {
		d.fileMode = mode
	}
}

func WithDirMode(mode fs.FileMode) Option {
	return func(d *Driver) {
		d.dirMode = mode
	}
}
