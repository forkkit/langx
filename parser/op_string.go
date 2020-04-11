// Code generated by "stringer -linecomment -type Op"; DO NOT EDIT.

package parser

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[OpNone-0]
	_ = x[OpModAsgn-1]
	_ = x[OpGe-2]
	_ = x[OpLe-3]
	_ = x[OpAnd-4]
	_ = x[OpOr-5]
	_ = x[OpEq-6]
	_ = x[OpNe-7]
	_ = x[OpAddAsgn-8]
	_ = x[OpSubAsgn-9]
	_ = x[OpMulAsgn-10]
	_ = x[OpDivAsgn-11]
	_ = x[OpPowAsgn-12]
	_ = x[OpSub-13]
	_ = x[OpAsgn-14]
	_ = x[OpAdd-15]
	_ = x[OpMul-16]
	_ = x[OpDiv-17]
	_ = x[OpLt-18]
	_ = x[OpGt-19]
	_ = x[OpMod-20]
	_ = x[OpPow-21]
	_ = x[OpNot-22]
	_ = x[OpSend-23]
}

const _Op_name = "%=>=<=&&||==!=+=-=*=/=^=-=+*/<>%^!->"

var _Op_index = [...]uint8{0, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 36}

func (i Op) String() string {
	if i < 0 || i >= Op(len(_Op_index)-1) {
		return "Op(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Op_name[_Op_index[i]:_Op_index[i+1]]
}
