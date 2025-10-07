package moExt

import "github.com/samber/mo"

func On[T any](input mo.Option[T], fn func(value T)) {
	if input.IsPresent() {
		fn(input.MustGet())
	}
}
