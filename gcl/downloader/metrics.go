// Copyright 2015 The go-gclchaineum Authors
// This file is part of the go-gclchaineum library.
//
// The go-gclchaineum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-gclchaineum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-gclchaineum library. If not, see <http://www.gnu.org/licenses/>.

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/gclchaineum/go-gclchaineum/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("gcl/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("gcl/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("gcl/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("gcl/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("gcl/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("gcl/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("gcl/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("gcl/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("gcl/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("gcl/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("gcl/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("gcl/downloader/receipts/timeout", nil)

	stateInMeter   = metrics.NewRegisteredMeter("gcl/downloader/states/in", nil)
	stateDropMeter = metrics.NewRegisteredMeter("gcl/downloader/states/drop", nil)
)
