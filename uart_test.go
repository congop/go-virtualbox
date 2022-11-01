package virtualbox

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

// StringProvider a function which returns a String
// Its String just returns the function call result
type StringProvider func() string

func (p StringProvider) String() string {
	return p()
}

// toVMInfoValueUARTn produces vm info value for uart<1-N> format <IO Base hex>,<IRQ> as  in uart1="0x03f8,4"
func (comConfig BasicSerialComConfig) toVMInfoValueUARTn() string {
	return fmt.Sprintf("0x%04x,%d", comConfig.Port, comConfig.IRQ)
}

func TestCanCreateFromOffUart(t *testing.T) {
	vmPropMap := map[string]string{"uart1": "off", "uart3": "off"}

	uarts, _ := NewUARTs(vmPropMap)

	expectedUARTs := &UARTs{
		UART1.UARTOffFromKey(), UART2.UARTOffFromKey(),
		UART3.UARTOffFromKey(), UART4.UARTOffFromKey(),
	}
	if diff := cmp.Diff(uarts, expectedUARTs); diff != "" {
		t.Errorf(
			`all uarts should have been off given uart1/3 are off
            	and others are not given; but
            	expected: %v
            	got     : %v
				diff    : %s
			`,
			expectedUARTs, uarts, diff)
	}
}

func TestUartSetAsUart1OffReturnCmdParamUart1Off(t *testing.T) {
	uart1Off := UART1.UARTOffFromKey()

	commands, err := uart1Off.commandParameters()
	commandsStr := fmt.Sprintf("%s", commands)
	expectedUart1Command := "--uart1 off"
	assert.Containsf(
		t, commandsStr, expectedUart1Command,
		`uart with state uart1=off did not return off command
			commands: %s
			expected in command : %s
			error    : %v`,
		commandsStr, expectedUart1Command, err)
}

func TestCanCreateFromUartWithVMInfoIOBaseIRQ(t *testing.T) {
	vmPropMap := map[string]string{"uart2": COM1().toVMInfoValueUARTn()}

	uarts, err := NewUARTs(vmPropMap)

	expectedUARTs := &[]UART{
		UART1.UARTOffFromKey(), {Key: UART2, ComConfig: COM1()},
		UART3.UARTOffFromKey(), UART4.UARTOffFromKey(),
	}
	assert.EqualValuesf(
		t, expectedUARTs, uarts,
		`all uarts should have been all off but COM1 at uart2
			expected: %v
			got     : %v
			diff    : %s
			error   : %v`,
		expectedUARTs, uarts, StringProvider(func() string { return cmp.Diff(uarts, expectedUARTs) }), err)
}

func TestUartSetAsUart3COM2ReturnCmdParamUart3WithCOM2Data(t *testing.T) {
	uart3Com3 := UART{Key: UART3, ComConfig: COM2()}

	commands, err := uart3Com3.commandParameters()

	expectedUart3Com2CommandParam := "--uart3 0x02f8 3"
	// commandsSorted := sort.StringSlice(commands)
	// commandsSorted.Sort()
	// i := commandsSorted.Search(expectedUart3Com2CommandParam)
	commandsStr := fmt.Sprintf("%s", commands)

	assert.Containsf(
		t, commandsStr, expectedUart3Com2CommandParam,
		`uart3 with com3 not in command params
			commands: %s
			expected in command : %s
			nerror    : %v`,
		commandsStr, expectedUart3Com2CommandParam, err)
}

func doTestNewUARTsCanCreateFromVMInfoMode(
	t *testing.T, vmInfoModeValue string, expectedMode UARTMode, expectedModeData string) {
	vmInfoMap := map[string]string{"uart3": "0x02f8,3", "uartmode3": vmInfoModeValue}

	uarts, err := NewUARTs(vmInfoMap)
	uartsExpected := &UARTs{
		UART1.UARTOffFromKey(), UART2.UARTOffFromKey(),
		{Key: UART3, Mode: expectedMode, ModeData: expectedModeData, ComConfig: COM2()},
		UART4.UARTOffFromKey(),
	}
	assert.EqualValuesf(
		t, uarts, uartsExpected,
		`uartmode3 should have been file with specified path
			err:`,
		err)
}

func TestNewUARTsCanCreateFromVMInfoMapFileMode(t *testing.T) {
	////--uartmode<1-N> file <file>
	// uartmode1="file,/tmp/ubuntu-focal-1"
	doTestNewUARTsCanCreateFromVMInfoMode(
		t, "file,/tmp/ubuntu-focal-1", UARTModeFile, "/tmp/ubuntu-focal-1")
}

func TestNewUARTsCanCreateFromVMInfoMapTcpClientMode(t *testing.T) {
	//--uartmode<1-N> tcpclient <hostname>:<port>
	// uartmode2="tcpclient,127.0.0.1:5555"
	doTestNewUARTsCanCreateFromVMInfoMode(
		t, "tcpclient,127.0.0.1:5555", UARTModeTCPClient, "127.0.0.1:5555")
}

func TestNewUARTsCanCreateFromVMInfoMapTcpServerMode(t *testing.T) {
	//--uartmode<1-N> tcpserver <port>
	//uartmode3="tcpserver,6666"
	doTestNewUARTsCanCreateFromVMInfoMode(
		t, "tcpserver,6666", UARTModeTCPServer, "6666")
}

func TestNewUARTsCanCreateFromVMInfoMapDisconnectedMode(t *testing.T) {
	//--uartmode<1-N> disconnected
	//uartmode3="disconnected"
	doTestNewUARTsCanCreateFromVMInfoMode(
		t, "disconnected", UARTModeDisconnected, "")
}

func TestNewUARTsCanCreateFromVMInfoMapServerMode(t *testing.T) {
	//--uartmode<1-N> server <pipe>
	//????
	t.SkipNow()
}

func TestNewUARTsCanCreateFromVMInfoMapClientMode(t *testing.T) {
	// --uartmode<1-N> client <pipe>
	//????
	t.SkipNow()
}

func TestNewUARTsCanCreateFromVMInfoMapUARTType(t *testing.T) {
	//uarttype4="16550A"
	vmInfoMap := map[string]string{"uart3": "0x2f8,3", "uarttype3": "16550A"}

	uarts, err := NewUARTs(vmInfoMap)
	uartsExpected := &UARTs{
		UART1.UARTOffFromKey(), UART2.UARTOffFromKey(),
		{Key: UART3, ComConfig: COM2(), Type: UARTT16550A},
		UART4.UARTOffFromKey(),
	}
	assert.EqualValuesf(
		t, uartsExpected, uarts,
		`uarttype3 should have been 16550A
			err:`,
		err)
}

func TestNewUARTsAllOffDoesReturnUARTs1_4AllOff(t *testing.T) {
	uarts := NewUARTsAllOff()

	expectedUARTs := &[]UART{
		UART1.UARTOffFromKey(), UART2.UARTOffFromKey(),
		UART3.UARTOffFromKey(), UART4.UARTOffFromKey(),
	}
	assert.EqualValuesf(
		t, expectedUARTs, uarts,
		`all 4 uarts  should have been off
			expected: %v
			got     : %v
			diff    : %s`,
		expectedUARTs, uarts, StringProvider(func() string { return cmp.Diff(uarts, expectedUARTs) }))
}

func TestNewUARTCanCreateFromValidInput(t *testing.T) {
	uart, err := NewUART("uart1", "16550A", "0x2f8", "3", "file", "/tmp/uart1")

	assert.Equal(
		t,
		&UART{Key: UART1, ComConfig: BasicSerialComConfig{IRQ: 3, Port: 0x02f8},
			Type: UARTT16550A, Mode: UARTModeFile, ModeData: "/tmp/uart1"},
		uart,
		"UART should have been successfully constructed: err=%s", err)
	if err != nil {
		t.Errorf("New Construction with valid data should have been error free: %s", err)
	}
}

func TestModifyVMCommandParametersReturnsCmpToSwitchAllOffForUARTAllOff(t *testing.T) {
	uarts := NewUARTsAllOff()

	params, err := uarts.ModifyVMCommandParameters()
	commandsStr := cmdNameValueToCmds(params)

	assert.ElementsMatchf(
		t,
		[]string{"--uart1 off", "--uart2 off", "--uart3 off", "--uart4 off"},
		commandsStr,
		"Should  have got params to switch off all uarts: actual=%s ", commandsStr)
	assert.NoError(t, err, "Gettin cmd params from all off UARTs should have been error free")

}

func cmdNameValueToCmds(cmdNameThenCmdValueList []string) (cmdList []string) {
	cmdList = make([]string, 0, len(cmdNameThenCmdValueList)/2)
	indexOfLastCmdName := len(cmdNameThenCmdValueList) - 1 - 1
	for i, j := 0, 0; i <= indexOfLastCmdName; i, j = i+2, j+1 {
		cmdName := cmdNameThenCmdValueList[i]
		cmdValue := cmdNameThenCmdValueList[i+1]
		cmdList = append(cmdList, cmdName+" "+cmdValue)
	}
	return
}

func TestModifyVMCommandParametersReturnsCmpToConfigUART2AndUART4AndRestOff(t *testing.T) {
	uart2, err := NewUART("uart2", "16550A", "0x2f8", "3", "file", "/tmp/uart1")
	assert.NoErrorf(t, err, "Fail to create uart2")
	uart4, err := NewUART("uart4", "16750", "0x3E8", "4", "disconnected", "")
	assert.NoErrorf(t, err, "Fail to create uart4")
	uarts, err := NewUARTsFromUARTMap(
		map[UARTKey]UART{
			UART2: *uart2,
			UART4: *uart4,
		})
	assert.NoError(t, err, "NewUARTsFromUARTMap failed")
	///
	params, err := uarts.ModifyVMCommandParameters()
	///
	paramsExpected := []string{
		"--uart1", "off",
		"--uart2", "0x02f8", "3", "--uartmode2", "file", "/tmp/uart1", "--uarttype2", "16550A",
		"--uart3", "off",
		"--uart4", "0x03e8", "4", "--uartmode4", "disconnected", "--uarttype4", "16750",
	}
	assert.ElementsMatchf(
		t, paramsExpected, params,
		"Should  have got params to creat uart2 and switch off the other uarts:"+
			"\nactual   = %s "+
			"\nexpected = %s",
		params, paramsExpected)
	assert.NoError(t, err, "Gettin cmd params from all off UARTs should have been error free")

}

func TestUARTsWithoutUARTHavingStateOff(t *testing.T) {
	uart4, err := NewUART("uart4", "16750", "0x3E8", "4", "disconnected", "")
	assert.NoErrorf(t, err, "Fail to create uart4")
	uart2, err := NewUART("uart2", "16750", "0x3E8", "3", "disconnected", "")
	assert.NoErrorf(t, err, "Fail to create uart2")
	tests := []struct {
		name  string
		uarts *UARTs
		want  *UARTs
	}{
		{
			name:  "UARTs with only UARTs having state Off should be trimmed to empty",
			uarts: NewUARTsAllOff(),
			want:  func() *UARTs { v := make(UARTs, 0, 4); return &v }(),
		},
		{
			name:  "UARTs should contains the UART with non <off> state",
			uarts: func() *UARTs { v := UARTs{UART1.UARTOffFromKey(), *uart2, UART2.UARTOffFromKey(), *uart4}; return &v }(),
			want:  func() *UARTs { v := UARTs{*uart2, *uart4}; return &v }(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.uarts.WithoutUARTHavingStateOff(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UARTs.WithoutUARTHavingStateOff() = %v, want %v", got, tt.want)
			}
		})
	}
}
