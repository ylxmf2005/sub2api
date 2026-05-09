package projection

import (
	"context"
	"hash/fnv"
	"path"
	"strings"
	"syscall"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type Node struct {
	fs.Inode
	backend Backend
	relPath string
	isDir   bool
}

var _ fs.InodeEmbedder = (*Node)(nil)
var _ fs.NodeLookuper = (*Node)(nil)
var _ fs.NodeGetattrer = (*Node)(nil)
var _ fs.NodeReaddirer = (*Node)(nil)
var _ fs.NodeOpener = (*Node)(nil)
var _ fs.NodeReader = (*Node)(nil)
var _ fs.NodeWriter = (*Node)(nil)
var _ fs.NodeSetattrer = (*Node)(nil)
var _ fs.NodeMkdirer = (*Node)(nil)
var _ fs.NodeCreater = (*Node)(nil)
var _ fs.NodeUnlinker = (*Node)(nil)
var _ fs.NodeRmdirer = (*Node)(nil)
var _ fs.NodeRenamer = (*Node)(nil)

func NewRootNode(backend Backend) *Node {
	return &Node{backend: backend, relPath: ".", isDir: true}
}

func (n *Node) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	childPath := joinProjectPath(n.relPath, name)
	stat, err := n.backend.Stat(ctx, childPath)
	if err != nil {
		return nil, errnoFromError(err)
	}
	fillEntry(out, stat)
	child := &Node{backend: n.backend, relPath: childPath, isDir: stat.IsDir}
	return n.NewInode(ctx, child, stableAttr(childPath, stat)), 0
}

func (n *Node) Getattr(ctx context.Context, _ fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	stat, err := n.backend.Stat(ctx, n.relPath)
	if err != nil {
		return errnoFromError(err)
	}
	fillAttr(&out.Attr, stat)
	out.SetTimeout(0)
	return 0
}

func (n *Node) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	resp, err := n.backend.List(ctx, n.relPath)
	if err != nil {
		return nil, errnoFromError(err)
	}
	entries := make([]fuse.DirEntry, 0, len(resp.Entries))
	for _, entry := range resp.Entries {
		entryPath := joinProjectPath(n.relPath, entry.Name)
		entries = append(entries, fuse.DirEntry{
			Name: entry.Name,
			Mode: modeTypeFrom(entry.IsDir),
			Ino:  inodeForPath(entryPath),
		})
	}
	return fs.NewListDirStream(entries), 0
}

func (n *Node) Open(context.Context, uint32) (fs.FileHandle, uint32, syscall.Errno) {
	return nil, 0, 0
}

func (n *Node) Read(ctx context.Context, _ fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	resp, err := n.backend.Read(ctx, n.relPath, off, int64(len(dest)))
	if err != nil {
		return nil, errnoFromError(err)
	}
	return fuse.ReadResultData(resp.Data), 0
}

func (n *Node) Write(ctx context.Context, _ fs.FileHandle, data []byte, off int64) (uint32, syscall.Errno) {
	resp, err := n.backend.Write(ctx, n.relPath, data, off, false)
	if err != nil {
		return 0, errnoFromError(err)
	}
	return uint32(resp.BytesWritten), 0
}

func (n *Node) Setattr(ctx context.Context, _ fs.FileHandle, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	var stat protocol.FileStatResponse
	var err error
	changed := false
	if size, ok := in.GetSize(); ok {
		stat, err = n.backend.Truncate(ctx, n.relPath, int64(size))
		if err != nil {
			return errnoFromError(err)
		}
		changed = true
	}
	if mode, ok := in.GetMode(); ok {
		stat, err = n.backend.Chmod(ctx, n.relPath, mode)
		if err != nil {
			return errnoFromError(err)
		}
		changed = true
	}
	if !changed {
		var err error
		stat, err = n.backend.Stat(ctx, n.relPath)
		if err != nil {
			return errnoFromError(err)
		}
	}
	fillAttr(&out.Attr, stat)
	return 0
}

func (n *Node) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	childPath := joinProjectPath(n.relPath, name)
	stat, err := n.backend.Mkdir(ctx, childPath, mode)
	if err != nil {
		return nil, errnoFromError(err)
	}
	fillEntry(out, stat)
	child := &Node{backend: n.backend, relPath: childPath, isDir: true}
	return n.NewInode(ctx, child, stableAttr(childPath, stat)), 0
}

func (n *Node) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	childPath := joinProjectPath(n.relPath, name)
	if _, err := n.backend.Write(ctx, childPath, nil, 0, true); err != nil {
		return nil, nil, 0, errnoFromError(err)
	}
	if mode != 0 {
		if chmodStat, err := n.backend.Chmod(ctx, childPath, mode); err == nil {
			fillEntry(out, chmodStat)
		} else {
			return nil, nil, 0, errnoFromError(err)
		}
	} else {
		fileStat, err := n.backend.Stat(ctx, childPath)
		if err != nil {
			return nil, nil, 0, errnoFromError(err)
		}
		fillEntry(out, fileStat)
	}
	child := &Node{backend: n.backend, relPath: childPath}
	return n.NewInode(ctx, child, fs.StableAttr{Mode: fuse.S_IFREG, Ino: inodeForPath(childPath)}), nil, 0, 0
}

func (n *Node) Unlink(ctx context.Context, name string) syscall.Errno {
	if err := n.backend.Remove(ctx, joinProjectPath(n.relPath, name), false); err != nil {
		return errnoFromError(err)
	}
	return 0
}

func (n *Node) Rmdir(ctx context.Context, name string) syscall.Errno {
	if err := n.backend.Remove(ctx, joinProjectPath(n.relPath, name), true); err != nil {
		return errnoFromError(err)
	}
	return 0
}

func (n *Node) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno {
	if flags != 0 {
		return syscall.ENOTSUP
	}
	parent, ok := newParent.(*Node)
	if !ok {
		if inode := newParent.EmbeddedInode(); inode != nil {
			parent, _ = inode.Operations().(*Node)
		}
	}
	if parent == nil {
		return syscall.EXDEV
	}
	if err := n.backend.Rename(ctx, joinProjectPath(n.relPath, name), joinProjectPath(parent.relPath, newName)); err != nil {
		return errnoFromError(err)
	}
	return 0
}

func joinProjectPath(base string, elem string) string {
	base = strings.TrimSpace(base)
	elem = strings.TrimSpace(elem)
	if base == "" || base == "." {
		if elem == "" {
			return "."
		}
		return path.Clean(elem)
	}
	if elem == "" {
		return path.Clean(base)
	}
	return path.Clean(path.Join(base, elem))
}

func fillEntry(out *fuse.EntryOut, stat protocol.FileStatResponse) {
	fillAttr(&out.Attr, stat)
	out.SetEntryTimeout(0)
	out.SetAttrTimeout(0)
}

func fillAttr(out *fuse.Attr, stat protocol.FileStatResponse) {
	out.Mode = stat.Mode
	if stat.IsDir {
		out.Mode = fuse.S_IFDIR | (stat.Mode & 0o7777)
	} else if out.Mode&syscall.S_IFMT == 0 {
		out.Mode = fuse.S_IFREG | (stat.Mode & 0o7777)
	}
	out.Size = uint64(max(stat.Size, 0))
	out.Mtime = uint64(stat.ModTime.Unix())
	out.Mtimensec = uint32(stat.ModTime.Nanosecond())
	out.Ctime = out.Mtime
	out.Ctimensec = out.Mtimensec
	out.Atime = out.Mtime
	out.Atimensec = out.Mtimensec
	out.Nlink = 1
	out.Ino = inodeForPath(stat.Path)
}

func stableAttr(projectPath string, stat protocol.FileStatResponse) fs.StableAttr {
	return fs.StableAttr{Mode: modeType(stat), Ino: inodeForPath(projectPath)}
}

func modeType(stat protocol.FileStatResponse) uint32 {
	return modeTypeFrom(stat.IsDir)
}

func modeTypeFrom(isDir bool) uint32 {
	if isDir {
		return fuse.S_IFDIR
	}
	return fuse.S_IFREG
}

func inodeForPath(projectPath string) uint64 {
	if projectPath == "" {
		projectPath = "."
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(projectPath))
	ino := h.Sum64()
	if ino == 0 || ino == ^uint64(0) {
		ino = 2
	}
	return ino
}
