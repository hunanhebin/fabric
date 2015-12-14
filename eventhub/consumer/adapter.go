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
	ehpb "github.com/openblockchain/obc-peer/eventhub/protos"
)

//EventAdapter is the interface by which a OBC Event Hub client registers interested events and 
//receives messages from the OBC Even Hub Server
type EventAdapter interface {
	GetInterestedEvents() ([]*ehpb.InterestedEvent)
	Recv(msg *ehpb.EventHubMessage) error
	Done(err error)
}
