package lock

import "errors"

var ErrNotHeld = errors.New("lock not held")
