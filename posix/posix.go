package posix // import "rs3.io/go/lua/posix"

import (
	"os"
	"os/user"
	"strconv"
	"syscall"

	lua "github.com/Shopify/go-lua"
	"golang.org/x/sys/unix"
)

// TODO(raynard): this should probably be per-lua.State
var errno syscall.Errno

func pushError(l *lua.State, err error) int {
	l.PushNil()
	l.PushString(err.Error())
	if x, ok := err.(syscall.Errno); ok {
		errno = x
	}
	l.PushInteger(int(errno))
	return 3
}

func access(l *lua.State) int {
	var err error
	name := lua.CheckString(l, 1)
	mode := lua.OptString(l, 2, "f")
	x := unix.F_OK
	for _, c := range mode {
		switch c {
		case 'f':
			x |= unix.F_OK
		case 'r':
			x |= unix.R_OK
		case 'w':
			x |= unix.W_OK
		case 'x':
			x |= unix.X_OK
		default:
			err = os.ErrInvalid
		}
	}
	if err == nil {
		if err = unix.Access(name, uint32(x)); err != nil {
			errno, _ = err.(syscall.Errno)
		} else {
			l.PushInteger(0)
			return 1
		}
	}
	return pushError(l, err)
}

func chown(l *lua.State) int {
	path := lua.CheckString(l, 1)
	lua.CheckAny(l, 2)
	lua.CheckAny(l, 3)

	uid, ok := l.ToInteger(2)
	if !ok {
		u, err := user.Lookup(lua.CheckString(l, 2))
		if err != nil {
			return pushError(l, syscall.ENOENT)
		}
		if uid, err = strconv.Atoi(u.Uid); err != nil {
			return pushError(l, err)
		}
	}

	gid, ok := l.ToInteger(3)
	if !ok {
		g, err := user.LookupGroup(lua.CheckString(l, 3))
		if err != nil {
			return pushError(l, syscall.ENOENT)
		}
		if gid, err = strconv.Atoi(g.Gid); err != nil {
			return pushError(l, err)
		}
	}

	if err := os.Chown(path, uid, gid); err != nil {
		pe, _ := err.(*os.PathError)
		return pushError(l, pe.Unwrap())
	}
	l.PushInteger(0)
	return 1
}

func getErrno(l *lua.State) int {
	l.PushInteger(int(errno))
	return 1
}

func linkFunction(f func(string, string) error) lua.Function {
	return func(l *lua.State) int {
		oldname := lua.CheckString(l, 1)
		newname := lua.CheckString(l, 2)
		if err := f(oldname, newname); err != nil {
			le, _ := err.(*os.LinkError)
			return pushError(l, le.Unwrap())
		}
		l.PushInteger(0)
		return 1
	}
}

var library = []lua.RegistryFunction{
	{Name: "access", Function: access},
	{Name: "chown", Function: chown},
	{Name: "errno", Function: getErrno},
	{Name: "link", Function: linkFunction(os.Link)},
	{Name: "symlink", Function: linkFunction(os.Symlink)},
}

func Open(l *lua.State) int {
	lua.NewLibrary(l, library)
	return 1
}
