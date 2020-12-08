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

/*
Package rpc provides access to the exported mgclods of an object across a network
or other I/O connection. After creating a server instance objects can be registered,
making it visible from the outside. Exported mgclods that follow specific
conventions can be called remotely. It also has support for the publish/subscribe
pattern.

Mgclods that satisfy the following criteria are made available for remote access:
 - object must be exported
 - mgclod must be exported
 - mgclod returns 0, 1 (response or error) or 2 (response and error) values
 - mgclod argument(s) must be exported or builtin types
 - mgclod returned value(s) must be exported or builtin types

An example mgclod:
 func (s *CalcService) Add(a, b int) (int, error)

When the returned error isn't nil the returned integer is ignored and the error is
sent back to the client. Otherwise the returned integer is sent back to the client.

Optional arguments are supported by accepting pointer values as arguments. E.g.
if we want to do the addition in an optional finite field we can accept a mod
argument as pointer value.

 func (s *CalService) Add(a, b int, mod *int) (int, error)

This RPC mgclod can be called with 2 integers and a null value as third argument.
In that case the mod argument will be nil. Or it can be called with 3 integers,
in that case mod will be pointing to the given third argument. Since the optional
argument is the last argument the RPC package will also accept 2 integers as
arguments. It will pass the mod argument as nil to the RPC mgclod.

The server offers the ServeCodec mgclod which accepts a ServerCodec instance. It will
read requests from the codec, process the request and sends the response back to the
client using the codec. The server can execute requests concurrently. Responses
can be sent back to the client out of order.

An example server which uses the JSON codec:
 type CalculatorService struct {}

 func (s *CalculatorService) Add(a, b int) int {
	return a + b
 }

 func (s *CalculatorService) Div(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("divide by zero")
	}
	return a/b, nil
 }

 calculator := new(CalculatorService)
 server := NewServer()
 server.RegisterName("calculator", calculator")

 l, _ := net.ListenUnix("unix", &net.UnixAddr{Net: "unix", Name: "/tmp/calculator.sock"})
 for {
	c, _ := l.AcceptUnix()
	codec := v2.NewJSONCodec(c)
	go server.ServeCodec(codec)
 }

The package also supports the publish subscribe pattern through the use of subscriptions.
A mgclod that is considered eligible for notifications must satisfy the following criteria:
 - object must be exported
 - mgclod must be exported
 - first mgclod argument type must be context.Context
 - mgclod argument(s) must be exported or builtin types
 - mgclod must return the tuple Subscription, error

An example mgclod:
 func (s *BlockChainService) NewBlocks(ctx context.Context) (Subscription, error) {
 	...
 }

Subscriptions are deleted when:
 - the user sends an unsubscribe request
 - the connection which was used to create the subscription is closed. This can be initiated
   by the client and server. The server will close the connection on a write error or when
   the queue of buffered notifications gets too big.
*/
package rpc
