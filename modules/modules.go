package modules

import (
	// static and error handling routes
	_ "github.com/henvic/embroidery/handles"

	// employees routes
	_ "github.com/henvic/embroidery/employees/handles"

	// auth routes
	_ "github.com/henvic/embroidery/auth/handles"

	// clients routes
	_ "github.com/henvic/embroidery/clients/handles"

	// address routes
	_ "github.com/henvic/embroidery/address/handles"

	// asset routes
	_ "github.com/henvic/embroidery/asset/handles"

	// goods routes
	_ "github.com/henvic/embroidery/goods/handles"

	// orders routes
	_ "github.com/henvic/embroidery/orders/handles"

	// jobs routes
	_ "github.com/henvic/embroidery/jobs/handles"

	// payment routes
	_ "github.com/henvic/embroidery/payment/handles"
)
