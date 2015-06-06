package util

import (
	"strings"
)

type Errors []error

func (errs Errors) Error() string {
	errstrs := make([]string, len(errs))
	for i, e := range errs {
		errstrs[i] = e.Error()
	}

	return "Errors: [" + strings.Join(errstrs, ", ") + "]"
}

func ToErrors(es []error) error {
	if len(es) == 0 {
		return nil
	}
	if len(es) == 1 {
		return es[0]
	}
	return Errors(es)
}
