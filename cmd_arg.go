package virtualbox

//CmdArg models a command arg, which can be a flag or not.
type CmdArg struct {
	K             string                     //Args key. e.g. --accelerated
	V             *string                    // V value. nil if arg does not allow value specification. Note that empty string "" is a valid value, and different from nil.
	ToCmdArgParts func(K, V string) []string //
	Del           bool                       // if true deleted, the arg will not be part of the final command
}

type CmdArgs struct {
	args      []CmdArg
	overrides []CmdArg
}

func NewCmdArgNoValue(k string) CmdArg {
	return CmdArg{K: k}
}

func NewCmdArg(k, v string) CmdArg {
	return CmdArg{K: k, V: &v}
}

func NewCmdArgDeleted(k string) CmdArg {
	return CmdArg{K: k, V: nil, Del: true}
}

func (cmdArgs *CmdArgs) AppendNoValue(key string) {
	cmdArgs.args = append(cmdArgs.args, NewCmdArgNoValue(key))
}

func (cmdArgs *CmdArgs) Append(key, value string) {
	cmdArgs.args = append(cmdArgs.args, NewCmdArg(key, value))
}

func (cmdArgs *CmdArgs) AppendCmdArgs(arg ...CmdArg) {
	if len(arg) == 0 {
		return
	}
	cmdArgs.args = append(cmdArgs.args, arg...)
}

func (cmdArgs *CmdArgs) AppendOverride(arg ...CmdArg) {
	if len(arg) == 0 {
		return
	}
	cmdArgs.overrides = append(cmdArgs.overrides, arg...)
}

//Args returns an slice containing the args which can be use in a command execution context.
//Args with multiple occurrence are not supported we will be overriding values.
func (cmdArgs CmdArgs) Args() []string {
	m := make(map[string]CmdArg, len(cmdArgs.args)+len(cmdArgs.overrides))
	orderK := make([]string, 0, len(cmdArgs.args)+len(cmdArgs.overrides))
	for _, curArgs := range [][]CmdArg{cmdArgs.args, cmdArgs.overrides} {
		for _, arg := range curArgs {
			_, contains := m[arg.K]
			if !contains {
				orderK = append(orderK, arg.K)
			}
			// yes we are always overriding --> multiple occurrences are not supported
			m[arg.K] = arg
		}
	}
	argStrs := make([]string, 0, len(orderK))
	for _, k := range orderK {
		arg := m[k]
		if arg.Del {
			continue
		}
		if arg.V == nil {
			argStrs = append(argStrs, arg.K)
		} else {
			if arg.ToCmdArgParts == nil {
				argStrs = append(argStrs, arg.K)
				argStrs = append(argStrs, *arg.V)
			} else {
				argStrs = append(argStrs, arg.ToCmdArgParts(arg.K, *arg.V)...)
			}
		}
	}
	return argStrs
}
