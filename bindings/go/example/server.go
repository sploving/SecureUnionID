// Copyright 2021 The SecureUnionID Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package example

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/protobuf/proto"
	"github.com/volcengine/SecureUnionID/bindings/go/core"
)

var (
	serverIdentity string
	skVersion      string
	sk             [core.BIGLENTH]byte
)

func SkToString(sk [core.BIGLENTH]byte) string {
	tmp := make([]byte, 0, core.BIGLENTH)
	for _, b := range sk {
		tmp = append(tmp, b)
	}
	return base64.StdEncoding.EncodeToString(tmp)
}

func StringToSk(skStr string) (sk [core.BIGLENTH]byte, err error) {
	b, err := base64.StdEncoding.DecodeString(skStr)
	if err != nil {
		return sk, err
	}
	for i, bd := range b {
		sk[i] = bd
	}
	return sk, nil
}

func DoEncryption(dids []string, skStr string) (vals []string, err error) {
	ski, err := StringToSk(skStr)
	if err != nil {
		return nil, err
	}
	vals = make([]string, 0, len(dids))
	server := core.NewSeverFromInput(ski)
	for _, did := range dids {
		hashdid := core.HashToG1(did)
		encDid, err := server.Enc(hashdid)
		if err != nil {
			return nil, err
		}
		vals = append(vals, encDid)
	}
	return vals, nil
}

func InitServer(receiver string, skStr string, skv string) {
	serverIdentity = receiver
	skVersion = skv
	skData, err := StringToSk(skStr)
	if err != nil || len(skData) != core.BIGLENTH {
		panic(err)
	}
	for i, b := range skData {
		sk[i] = b
	}
}

func psiEccEncode(ski [core.BIGLENTH]byte, msgs []string) (encodedMsgs []string, err error) {
	server := core.NewSeverFromInput(ski)
	for _, msg := range msgs {
		resMsg, err := server.Enc(msg)
		if err != nil {
			return nil, err
		}
		fmt.Println("msg: ", msg, "resMsg: ", resMsg)
		encodedMsgs = append(encodedMsgs, resMsg)
	}
	return encodedMsgs, nil
}

func serverEncode(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}
	signReq := &SignRequest{}
	err = proto.Unmarshal(b, signReq)
	if err != nil {
		return
	}
	resMsgs, err := psiEccEncode(sk, signReq.GetBlindedMessages())
	rspCode := ResponseCode_Success
	if err != nil {
		rspCode = ResponseCode_InnerAlgorithmFailed
	}
	signRsp := &SignResponse{
		Receiver:       &serverIdentity,
		RspCode:        &rspCode,
		SkVersion:      &skVersion,
		SignedMessages: resMsgs,
	}
	rb, err := proto.Marshal(signRsp)
	if err != nil {
		return
	}
	w.Write(rb)
	fmt.Println("dsp: ", signReq.GetSender(), " called, server success return")
}

func StartServer() {
	fmt.Println("Start Server")
	http.HandleFunc("/encryption", serverEncode)
	http.ListenAndServe(":8008", nil)
}
