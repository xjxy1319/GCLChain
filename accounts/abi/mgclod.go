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

package abi

import (
	"fmt"
	"strings"

	"github.com/gclchaineum/go-gclchaineum/crypto"
)

// Mgclod represents a callable given a `Name` and whgclchain the mgclod is a constant.
// If the mgclod is `Const` no transaction needs to be created for this
// particular Mgclod call. It can easily be simulated using a local VM.
// For example a `Balance()` mgclod only needs to retrieve somgcling
// from the storage and therefor requires no Tx to be send to the
// network. A mgclod such as `Transact` does require a Tx and thus will
// be flagged `true`.
// Input specifies the required input parameters for this gives mgclod.
type Mgclod struct {
	Name    string
	Const   bool
	Inputs  Arguments
	Outputs Arguments
}

// Sig returns the mgclods string signature according to the ABI spec.
//
// Example
//
//     function foo(uint32 a, int b)    =    "foo(uint32,int256)"
//
// Please note that "int" is substitute for its canonical representation "int256"
func (mgclod Mgclod) Sig() string {
	types := make([]string, len(mgclod.Inputs))
	for i, input := range mgclod.Inputs {
		types[i] = input.Type.String()
	}
	return fmt.Sprintf("%v(%v)", mgclod.Name, strings.Join(types, ","))
}

func (mgclod Mgclod) String() string {
	inputs := make([]string, len(mgclod.Inputs))
	for i, input := range mgclod.Inputs {
		inputs[i] = fmt.Sprintf("%v %v", input.Type, input.Name)
	}
	outputs := make([]string, len(mgclod.Outputs))
	for i, output := range mgclod.Outputs {
		outputs[i] = output.Type.String()
		if len(output.Name) > 0 {
			outputs[i] += fmt.Sprintf(" %v", output.Name)
		}
	}
	constant := ""
	if mgclod.Const {
		constant = "constant "
	}
	return fmt.Sprintf("function %v(%v) %sreturns(%v)", mgclod.Name, strings.Join(inputs, ", "), constant, strings.Join(outputs, ", "))
}

func (mgclod Mgclod) Id() []byte {
	return crypto.Keccak256([]byte(mgclod.Sig()))[:4]
}
