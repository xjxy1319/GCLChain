// Copyright 2018 The go-gclchaineum Authors
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

package rawdb

// DatabaseReader wraps the Has and Get mgclod of a backing data store.
type DatabaseReader interface {
	Has(key []byte) (bool, error)
	Get(key []byte) ([]byte, error)
}

// DatabaseWriter wraps the Put mgclod of a backing data store.
type DatabaseWriter interface {
	Put(key []byte, value []byte) error
}

// DatabaseDeleter wraps the Delete mgclod of a backing data store.
type DatabaseDeleter interface {
	Delete(key []byte) error
}
