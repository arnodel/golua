//go:build noquotas
// +build noquotas

package pattern

// countSteps says whether a match counts its steps and charges them.  Without
// quotas nothing would use the count, so the code that keeps it is compiled
// out.
const countSteps = false
