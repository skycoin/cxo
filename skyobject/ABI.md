Binary data format
==================

# Data

## Int, Uint

All signed and unsigned integers encoded using little-endian order to
bytes slice with appropriate size

- int8, uint8 - one byte
- int16, uint16 - 2 byte
- int32, uint32 - 4 byte
- int64, uint64 - 8 byte

That is obvious.

See [LittleEndian](http://godoc.org/encoding/binary#LittleEndian) and
[ByteOrder](http://godoc.org/encoding/binary#ByteOrder)

## Float32, Flaot64

For float point number used IEEE 754 binary representation.
E.g. floatXX -> IEEE -> uintXX -> little-endian -> bytes

- float32 - 4 byte
- float64 - 8 byte

See [`math.Float32bits`](http://godoc.org/math#Float32bits),
[`math.Float32frombits`](http://godoc.org/math#Float32frombits) and
[`math.Float64bits`](http://godoc.org/math#Float64bits),
[`math.Float64frombits`](http://godoc.org/math#Float64frombits) methods

## String

Strings are length-prefixed, where length is represented as encoded `uint32`

```
[4 byte length][string body or nothing if the length is zero]
```

## Arrays

Arrays is just encoded elements one by one

## Slices

Slices are length-prefixed like strings

```
[4 byte length][elements one by one]
```

## Structures

Structures is encoded elements from first field to last

For example

```go
type Age uint32

type User strut {
	Age  Age
	Name string
}
```

should be encoded to

```
[4 byte of the Age]
[4 byte of the Name length]
[the name in person]
```

That's all about data

# References

Since we can't use pointers, we are using SHA256 hash of encoded value.
This can be applied only to structures. (E.g. only "pointers" to structures).
There are 3 types of references

# Schema

...
