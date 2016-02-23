package vmdkops

type VolumeError int

const (
	NotFound VolumeError = iota
)

func (e *VolumeError) Error() string {
	switch *e {
	case NotFound:
		return "Volume does not exist"
	}
	// Is this unreachable?
	return ""
}
