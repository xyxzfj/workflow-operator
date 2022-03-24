package reconciler

import ctrl "sigs.k8s.io/controller-runtime"

// NewReturn
func NewReturn(result ctrl.Result, err error) *Return {
	return &Return{
		result: result,
		err:    err,
	}
}

// Return
type Return struct {
	result ctrl.Result
	err    error
}

// Error
func (r *Return) Error() string {
	return r.err.Error()
}

// Return
func (r *Return) Return() (ctrl.Result, error) {
	return r.result, r.err
}
