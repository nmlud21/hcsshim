//go:build windows

package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	specs "github.com/opencontainers/runtime-spec/specs-go"

	"github.com/Microsoft/go-winio"
	"github.com/Microsoft/go-winio/pkg/guid"
	"github.com/Microsoft/hcsshim/internal/gcs"
)

func acceptAndClose(l net.Listener) (conn net.Conn, err error) {
	conn, err = l.Accept()
	l.Close()

	return conn, err
}

func validatePolicy(mountPolicy specs.Mount, checkPolicy specs.Mount, file os.File) bool {
	if mountPolicy.Destination != checkPolicy.Destination {
		_, err := file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - Mount policy denied: expected: " + mountPolicy.Destination + ", got: " + checkPolicy.Destination + "\n")
		if err != nil {
			fmt.Printf("Error writing to file: %v", err)
			return false
		}
		return false
	} else if mountPolicy.Type != checkPolicy.Type {
		_, err := file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - Mount policy denied: expected: " + mountPolicy.Type + ", got: " + checkPolicy.Type + "\n")
		if err != nil {
			fmt.Printf("Error writing to file: %v", err)
			return false
		}
		return false
	} else if mountPolicy.Source != checkPolicy.Source {
		_, err := file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - Mount policy denied: expected: " + mountPolicy.Source + ", got: " + checkPolicy.Source + "\n")
		if err != nil {
			fmt.Printf("Error writing to file: %v", err)
			return false
		}
		return false
	}

	return true
}

func main() {
	ctx := context.Background()
	file, err := os.OpenFile("sidecar.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Error opening file: %v", err)
		return
	}
	defer file.Close()

	d := &winio.HvsockDialer{
		Deadline:  time.Now().Add(10 * time.Minute),
		Retries:   1000,
		RetryWait: time.Second,
	}

	//DEFINE_GUID(HV_GUID_PARENT, 0xa42e7cda, 0xd03f, 0x480c, 0x9c, 0xc2, 0xa4, 0xde, 0x20, 0xab, 0xb8, 0x78);
	addr := &winio.HvsockAddr{
		VMID: guid.GUID{
			Data1: 0xa42e7cda,
			Data2: 0xd03f,
			Data3: 0x480c,
			Data4: [8]uint8{0x9c, 0xc2, 0xa4, 0xde, 0x20, 0xab, 0xb8, 0x78},
		},
		ServiceID: gcs.SidecarGcsHvsockServiceID,
	}

	_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - started dialing\n")
	if err != nil {
		fmt.Printf("Error writing to file: %v", err)
		return
	}

	// Dial the HV socket for the sidecar gcs
	conn, err := d.Dial(ctx, addr)
	if err != nil {
		_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - Error dialing address: " + err.Error() + "\n")
		if err != nil {
			fmt.Printf("Error writing to file: %v", err)
			return
		}
		return
	}

	go func() {
		_, err := file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - Connection established\n")
		if err != nil {
			fmt.Printf("Error writing to file: %v", err)
			return
		}
		// for {
		// 	_, err := conn.Write([]byte(time.Now().Format("2006-01-02 15:04:05") + " - Message from sidecar\n"))
		// 	if err != nil {
		// 		_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - Error writing to connection: " + err.Error() + "\n")
		// 		if err != nil {
		// 			fmt.Printf("Error writing to file: %v", err)
		// 			return
		// 		}
		// 		return
		// 	}

		// 	time.Sleep(5 * time.Second)
		// }

		mountPolicy := specs.Mount{
			Destination: "C:\\dest\\mount\\path",
			Type:        "physical-disk",
			Source:      "C:\\src\\mount\\path",
		}

		var denyPolicy1 specs.Mount
		var denyPolicy2 specs.Mount
		var acceptPolicy specs.Mount
		dec := gob.NewDecoder(conn)
		err = dec.Decode(&denyPolicy1)
		if err != nil {
			_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - Error decoding mount policy\n")
			if err != nil {
				fmt.Printf("Error writing to file: %v", err)
				return
			}
			return
		}

		err = dec.Decode(&denyPolicy2)
		if err != nil {
			_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - Error decoding mount policy\n")
			if err != nil {
				fmt.Printf("Error writing to file: %v", err)
				return
			}
			return
		}

		err = dec.Decode(&acceptPolicy)
		if err != nil {
			_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - Error decoding mount policy\n")
			if err != nil {
				fmt.Printf("Error writing to file: %v", err)
				return
			}
			return
		}

		_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - Deny Policy - " + denyPolicy1.Destination + ", " + denyPolicy1.Type + ", " + denyPolicy1.Source + " read successfully\n")
		if err != nil {
			fmt.Printf("Error writing to file: %v", err)
			return
		}

		_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - Accept Policy - " + acceptPolicy.Destination + ", " + acceptPolicy.Type + ", " + acceptPolicy.Source + " read successfully\n")
		if err != nil {
			fmt.Printf("Error writing to file: %v", err)
			return
		}

		//Check against the mount policy
		deny1IsValid := validatePolicy(mountPolicy, denyPolicy1, *file)
		deny1IsValidStr := strconv.FormatBool(deny1IsValid)
		deny2IsValid := validatePolicy(mountPolicy, denyPolicy2, *file)
		deny2IsValidStr := strconv.FormatBool(deny2IsValid)
		acceptIsValid := validatePolicy(mountPolicy, acceptPolicy, *file)
		acceptIsValidStr := strconv.FormatBool(acceptIsValid)
		_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - Deny 1 Policy: expected: false; actual: " + deny1IsValidStr + "\n")
		if err != nil {
			fmt.Printf("Error writing to file: %v", err)
			return
		}

		_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - Deny 2 Policy: expected: false; actual: " + deny2IsValidStr + "\n")
		if err != nil {
			fmt.Printf("Error writing to file: %v", err)
			return
		}

		_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - Accept Policy: expected: true; actual: " + acceptIsValidStr + "\n")
		if err != nil {
			fmt.Printf("Error writing to file: %v", err)
			return
		}

		enc := gob.NewEncoder(conn)
		err = enc.Encode(deny1IsValid)
		if err != nil {
			_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - Error encoding mount policy\n")
			if err != nil {
				fmt.Printf("Error writing to file: %v", err)
				return
			}
			return
		}

		err = enc.Encode(deny2IsValid)
		if err != nil {
			_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - Error encoding mount policy\n")
			if err != nil {
				fmt.Printf("Error writing to file: %v", err)
				return
			}
			return
		}

		err = enc.Encode(acceptIsValid)
		if err != nil {
			_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - Error encoding mount policy\n")
			if err != nil {
				fmt.Printf("Error writing to file: %v", err)
				return
			}
			return
		}

	}()

	///////////////////////////////////////////////
	// GCS connection
	///////////////////////////////////////////////

	// start GCS listener
	// 0xe0e16197, 0xdd56, 0x4a10, 0x91, 0x95, 0x5e, 0xe7, 0xa1, 0x55, 0xa8, 0x38
	l, err := winio.ListenHvsock(&winio.HvsockAddr{
		VMID: guid.GUID{
			Data1: 0xe0e16197,
			Data2: 0xdd56,
			Data3: 0x4a10,
			Data4: [8]uint8{0x91, 0x95, 0x5e, 0xe7, 0xa1, 0x55, 0xa8, 0x38},
		},
		ServiceID: gcs.SidecarGuestHvsockServiceID,
	})
	if err != nil {
		_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - Error opening gcs listener: " + err.Error() + "\n")
		if err != nil {
			fmt.Printf("Error writing to file: %v", err)
			return
		}
		return
	}

	if l != nil {
		_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - GCS listener opened\n")
		if err != nil {
			fmt.Printf("Error writing to file: %v", err)
			return
		}

		// Accept the connection
		gcsConn, err := acceptAndClose(l)
		if err != nil {
			_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - Error with accept: " + err.Error() + "\n")
			if err != nil {
				fmt.Printf("Error writing to file: %v", err)
				return
			}
			return
		}

		_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - GCS connection established\n")
		if err != nil {
			fmt.Printf("Error writing to file: %v", err)
			return
		}

		l = nil

		go func() {
			logfile, err := os.OpenFile("gcsLog.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - error reading \n")
				if err != nil {
					return
				}
				return
			}
			defer logfile.Close()

			gcsbuffer := make([]byte, 1024)
			for {
				_, err := gcsConn.Read(gcsbuffer)
				if err != nil {
					_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - error reading from client\n")
					if err != nil {
						return
					}
					return
				}
				time.Sleep(5 * time.Second)

				if _, err := logfile.Write(gcsbuffer); err != nil {
					_, err = file.WriteString(time.Now().Format("2006-01-02 15:04:05") + " - error writing to logfile\n")
					if err != nil {
						return
					}
					return
				}
			}
		}()

		for {
			fmt.Println("Running sidecar")
		}
	}
}

//sidecar is client for shim, gcs is client for sidecar
