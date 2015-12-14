/*
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/

package consumer

import (
	"fmt"
	"io"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	"github.com/spf13/viper"

	ehpb "github.com/openblockchain/obc-peer/eventhub/protos"
)

type OBCEventClient struct {
	peerAddress  string
	stream ehpb.EventHub_ChatClient
	adapter EventAdapter
}

const defaultTimeout = time.Second * 3

// NewEventHubClientConnection Returns a new grpc.ClientConn to the configured local PEER.
func NewOBCEventHubClient(peerAddress string, adapter EventAdapter) *OBCEventClient {
	return &OBCEventClient{peerAddress, nil, adapter}
}

// NewEventHubClientConnectionWithAddress Returns a new grpc.ClientConn to the configured local PEER.
func newEventHubClientConnectionWithAddress(peerAddress string) (*grpc.ClientConn,error) {
	var opts []grpc.DialOption
	if viper.GetBool("peer.tls.enabled") {
		var sn string
		if viper.GetString("peer.tls.server-host-override") != "" {
			sn = viper.GetString("peer.tls.server-host-override")
		}
		var creds credentials.TransportAuthenticator
		if viper.GetString("peer.tls.cert.file") != "" {
			var err error
			creds, err = credentials.NewClientTLSFromFile(viper.GetString("peer.tls.cert.file"), sn)
			if err != nil {
				grpclog.Fatalf("Failed to create TLS credentials %v", err)
			}
		} else {
			creds = credentials.NewClientTLSFromCert(nil, sn)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	}
	opts = append(opts, grpc.WithTimeout(defaultTimeout))
	opts = append(opts, grpc.WithBlock())
	opts = append(opts, grpc.WithInsecure())

	return grpc.Dial(peerAddress, opts...)
}


func (ec *OBCEventClient) register(ies []*ehpb.InterestedEvent) error {
	emsg := &ehpb.EventHubMessage{&ehpb.EventHubMessage_RegisterEvent{&ehpb.RegisterEvent{ies}}}
	var err error
	if err = ec.stream.Send(emsg); err != nil {
		fmt.Printf("error on Register send %s\n", err)
		return err
	} 

	regChan := make(chan struct{})
	go func() {
		defer close(regChan)
		in, inerr := ec.stream.Recv()
		if inerr != nil {
			err = inerr
			return
		}
		switch in.Event.(type) {
		case *ehpb.EventHubMessage_RegisterEvent:
		case *ehpb.EventHubMessage_TransactionEvent:
			err = fmt.Errorf("invalid Transaction object for register")
		case nil:
			err = fmt.Errorf("invalid nil object for register")
		default:
			err = fmt.Errorf("invalid registration object")
		}
		regChan <- struct{}{}
	}()
	select {
	case  <- regChan:
	case <-time.After(5*time.Second):
		err = fmt.Errorf("timeout waiting for registration")
	}
	return err
}

func (ec *OBCEventClient) processEvents () error {
	defer ec.stream.CloseSend()
	for {
		in, err := ec.stream.Recv()
		if err == io.EOF {
			// read done.
			if ec.adapter != nil {
				ec.adapter.Done(nil)
			}
			return nil
		}
		if err != nil {
			if ec.adapter != nil {
				ec.adapter.Done(err)
			}
			return err
		}
		if ec.adapter != nil {
			err = ec.adapter.Recv(in)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (ec *OBCEventClient) Start() error {
	conn, err := newEventHubClientConnectionWithAddress(ec.peerAddress)
	if err != nil {
		return fmt.Errorf("Could not create client conn to %s", ec.peerAddress)
	}

	ies := ec.adapter.GetInterestedEvents()
	if ies == nil {
		return fmt.Errorf("no interested events")
	}

	serverClient := ehpb.NewEventHubClient(conn)
	ec.stream, err = serverClient.Chat(context.Background())
	if err != nil {
		return fmt.Errorf("Could not create client conn to %s", ec.peerAddress)
	}

	if err = ec.register(ies); err != nil {
		return err
	}

	go ec.processEvents()

	return nil
}

func (ec *OBCEventClient) Stop() error {
	return ec.stream.CloseSend()
}
