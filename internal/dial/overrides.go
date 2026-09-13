package dial

// Overrides are per-dial CLI values that beat the profile's stored defaults.
// Nil pointers mean "not overridden on this dial".
type Overrides struct {
	SessionWait  *int
	Verbosity    *int
	CallDuration *int
	RingTime     *int
	ColorOutput  *bool
	Method       *string
}