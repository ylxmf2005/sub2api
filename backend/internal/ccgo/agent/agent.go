package agent

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
)

type Agent struct {
	root  LocalRoot
	files *FileService
	exec  *ExecService
}

func New(rootPath string) (*Agent, error) {
	root, err := NewLocalRoot(rootPath)
	if err != nil {
		return nil, err
	}
	return NewWithRoot(root), nil
}

func NewWithRoot(root LocalRoot) *Agent {
	return &Agent{
		root:  root,
		files: NewFileService(root),
		exec:  NewExecService(root),
	}
}

func (a *Agent) Root() string {
	if a == nil {
		return ""
	}
	return a.root.Path()
}

func (a *Agent) Handle(ctx context.Context, env protocol.Envelope) (protocol.Envelope, error) {
	if a == nil {
		return protocol.Envelope{}, protocol.NewError(protocol.ErrorInternal, "agent is not configured")
	}
	if env.Type != protocol.MessageTypeRequest {
		return protocol.Envelope{}, protocol.NewError(protocol.ErrorInvalidRequest, "agent only handles request envelopes")
	}
	var payload any
	switch env.Method {
	case protocol.MethodFileStat:
		req, err := protocol.DecodePayload[protocol.FileStatRequest](env)
		if err != nil {
			return protocol.Envelope{}, err
		}
		resp, err := a.files.Stat(ctx, req)
		if err != nil {
			return protocol.NewErrorResponse(env.RequestID, protocolError(err)), nil
		}
		payload = resp
	case protocol.MethodFileRead:
		req, err := protocol.DecodePayload[protocol.FileReadRequest](env)
		if err != nil {
			return protocol.Envelope{}, err
		}
		resp, err := a.files.Read(ctx, req)
		if err != nil {
			return protocol.NewErrorResponse(env.RequestID, protocolError(err)), nil
		}
		payload = resp
	case protocol.MethodFileWrite:
		req, err := protocol.DecodePayload[protocol.FileWriteRequest](env)
		if err != nil {
			return protocol.Envelope{}, err
		}
		resp, err := a.files.Write(ctx, req)
		if err != nil {
			return protocol.NewErrorResponse(env.RequestID, protocolError(err)), nil
		}
		payload = resp
	case protocol.MethodFileList:
		req, err := protocol.DecodePayload[protocol.FileListRequest](env)
		if err != nil {
			return protocol.Envelope{}, err
		}
		resp, err := a.files.List(ctx, req)
		if err != nil {
			return protocol.NewErrorResponse(env.RequestID, protocolError(err)), nil
		}
		payload = resp
	case protocol.MethodFileMkdir:
		req, err := protocol.DecodePayload[protocol.FileMkdirRequest](env)
		if err != nil {
			return protocol.Envelope{}, err
		}
		resp, err := a.files.Mkdir(ctx, req)
		if err != nil {
			return protocol.NewErrorResponse(env.RequestID, protocolError(err)), nil
		}
		payload = resp
	case protocol.MethodFileRemove:
		req, err := protocol.DecodePayload[protocol.FileRemoveRequest](env)
		if err != nil {
			return protocol.Envelope{}, err
		}
		if err := a.files.Remove(ctx, req); err != nil {
			return protocol.NewErrorResponse(env.RequestID, protocolError(err)), nil
		}
		payload = map[string]bool{"ok": true}
	case protocol.MethodFileRename:
		req, err := protocol.DecodePayload[protocol.FileRenameRequest](env)
		if err != nil {
			return protocol.Envelope{}, err
		}
		if err := a.files.Rename(ctx, req); err != nil {
			return protocol.NewErrorResponse(env.RequestID, protocolError(err)), nil
		}
		payload = map[string]bool{"ok": true}
	case protocol.MethodFileTruncate:
		req, err := protocol.DecodePayload[protocol.FileTruncateRequest](env)
		if err != nil {
			return protocol.Envelope{}, err
		}
		resp, err := a.files.Truncate(ctx, req)
		if err != nil {
			return protocol.NewErrorResponse(env.RequestID, protocolError(err)), nil
		}
		payload = resp
	case protocol.MethodFileChmod:
		req, err := protocol.DecodePayload[protocol.FileChmodRequest](env)
		if err != nil {
			return protocol.Envelope{}, err
		}
		resp, err := a.files.Chmod(ctx, req)
		if err != nil {
			return protocol.NewErrorResponse(env.RequestID, protocolError(err)), nil
		}
		payload = resp
	case protocol.MethodExec:
		req, err := protocol.DecodePayload[protocol.ExecRequest](env)
		if err != nil {
			return protocol.Envelope{}, err
		}
		resp, err := a.exec.Run(ctx, req)
		if err != nil {
			return protocol.NewErrorResponse(env.RequestID, protocolError(err)), nil
		}
		payload = resp
	default:
		return protocol.NewErrorResponse(env.RequestID, protocol.NewError(protocol.ErrorUnsupported, "unsupported ccgo method")), nil
	}
	return protocol.NewResponse(env.RequestID, payload)
}

func protocolError(err error) *protocol.Error {
	if err == nil {
		return nil
	}
	if ccgoErr, ok := err.(*protocol.Error); ok {
		return ccgoErr
	}
	return protocol.NewError(protocol.ErrorInternal, err.Error())
}
