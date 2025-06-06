/*
 * Teleport
 * Copyright (C) 2025  Gravitational, Inc.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package auth

import (
	"context"
	"crypto"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/gravitational/trace"
	"github.com/stretchr/testify/require"

	"github.com/gravitational/teleport/api/client"
	"github.com/gravitational/teleport/api/client/proto"
	headerv1 "github.com/gravitational/teleport/api/gen/proto/go/teleport/header/v1"
	machineidv1pb "github.com/gravitational/teleport/api/gen/proto/go/teleport/machineid/v1"
	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/teleport/lib/auth/testauthority"
	"github.com/gravitational/teleport/lib/boundkeypair"
	"github.com/gravitational/teleport/lib/cryptosuites"
	"github.com/gravitational/teleport/lib/sshutils"
)

type mockBoundKeypairValidator struct {
	subject     string
	clusterName string
	publicKey   crypto.PublicKey
}

func (v *mockBoundKeypairValidator) IssueChallenge() (*boundkeypair.ChallengeDocument, error) {
	return &boundkeypair.ChallengeDocument{
		Nonce: "fake",
	}, nil
}

func (v *mockBoundKeypairValidator) ValidateChallengeResponse(issued *boundkeypair.ChallengeDocument, compactResponse string) error {
	// For testing, the solver will just reply with the marshaled public key, so
	// we'll parse and compare it.
	key, err := sshutils.CryptoPublicKey([]byte(compactResponse))
	if err != nil {
		return trace.Wrap(err, "parsing bound public key")
	}

	equal, ok := v.publicKey.(interface {
		Equal(x crypto.PublicKey) bool
	})
	if !ok {
		return trace.BadParameter("unsupported public key type %T", key)
	}

	if !equal.Equal(key) {
		return trace.AccessDenied("incorrect public key")
	}

	return nil
}

func testBoundKeypair(t *testing.T) (crypto.Signer, string) {
	key, err := cryptosuites.GeneratePrivateKeyWithAlgorithm(cryptosuites.ECDSAP256)
	require.NoError(t, err)

	return key.Signer, string(key.MarshalSSHPublicKey())
}

// parseJoinState parses a join state token without verification, for testing
// purposes only.
func parseJoinState(t *testing.T, state []byte) *boundkeypair.JoinState {
	token, err := jwt.ParseSigned(string(state))
	require.NoError(t, err)

	var doc boundkeypair.JoinState
	require.NoError(t, token.UnsafeClaimsWithoutVerification(&doc))

	return &doc
}

func TestServer_RegisterUsingBoundKeypairMethod(t *testing.T) {
	ctx := context.Background()

	_, correctPublicKey := testBoundKeypair(t)
	_, incorrectPublicKey := testBoundKeypair(t)

	srv := newTestTLSServer(t)
	auth := srv.Auth()
	auth.createBoundKeypairValidator = func(subject, clusterName string, publicKey crypto.PublicKey) (boundKeypairValidator, error) {
		return &mockBoundKeypairValidator{
			subject:     subject,
			clusterName: clusterName,
			publicKey:   publicKey,
		}, nil
	}

	_, err := CreateRole(ctx, auth, "example", types.RoleSpecV6{})
	require.NoError(t, err)

	adminClient, err := srv.NewClient(TestAdmin())
	require.NoError(t, err)

	_, err = adminClient.BotServiceClient().CreateBot(ctx, &machineidv1pb.CreateBotRequest{
		Bot: &machineidv1pb.Bot{
			Kind:    types.KindBot,
			Version: types.V1,
			Metadata: &headerv1.Metadata{
				Name: "test",
			},
			Spec: &machineidv1pb.BotSpec{
				Roles: []string{"example"},
			},
		},
	})
	require.NoError(t, err)

	sshPrivateKey, sshPublicKey, err := testauthority.New().GenerateKeyPair()
	require.NoError(t, err)
	tlsPublicKey, err := PrivateKeyToPublicKeyTLS(sshPrivateKey)
	require.NoError(t, err)

	jwtCA, err := auth.GetCertAuthority(ctx, types.CertAuthID{
		Type:       types.BoundKeypairCA,
		DomainName: srv.ClusterName(),
	}, /* loadKeys */ true)
	require.NoError(t, err)

	jwtSigner, err := auth.GetKeyStore().GetJWTSigner(ctx, jwtCA)
	require.NoError(t, err)

	// An invalid signer for signing "fake" JWTs.
	invalidJWTSigner, _ := testBoundKeypair(t)

	makeToken := func(mutators ...func(v2 *types.ProvisionTokenV2)) types.ProvisionTokenV2 {
		token := types.ProvisionTokenV2{
			Spec: types.ProvisionTokenSpecV2{
				JoinMethod: types.JoinMethodBoundKeypair,
				Roles:      []types.SystemRole{types.RoleBot},
				BotName:    "test",
				BoundKeypair: &types.ProvisionTokenSpecV2BoundKeypair{
					Onboarding: &types.ProvisionTokenSpecV2BoundKeypair_OnboardingSpec{
						InitialPublicKey: correctPublicKey,
					},
					Recovery: &types.ProvisionTokenSpecV2BoundKeypair_RecoverySpec{
						// Only insecure is supported for now.
						Mode: boundkeypair.RecoveryModeInsecure,
					},
				},
			},
			Status: &types.ProvisionTokenStatusV2{
				BoundKeypair: &types.ProvisionTokenStatusV2BoundKeypair{},
			},
		}
		for _, mutator := range mutators {
			mutator(&token)
		}
		return token
	}

	withRecovery := func(mode string, count, limit uint32, botInstanceID string) func(*types.ProvisionTokenV2) {
		return func(v2 *types.ProvisionTokenV2) {
			v2.Spec.BoundKeypair.Recovery.Mode = mode
			v2.Spec.BoundKeypair.Recovery.Limit = limit
			v2.Status.BoundKeypair.RecoveryCount = count
			v2.Status.BoundKeypair.BoundBotInstanceID = botInstanceID
		}
	}

	makeJoinState := func(signer crypto.Signer, mutators ...func(s *boundkeypair.JoinStateParams)) string {
		params := &boundkeypair.JoinStateParams{
			Clock:       srv.Clock(),
			ClusterName: srv.ClusterName(),
		}

		for _, mutator := range mutators {
			mutator(params)
		}

		state, err := boundkeypair.IssueJoinState(signer, params)
		require.NoError(t, err)

		return state
	}

	withToken := func(mutators ...func(v2 *types.ProvisionTokenV2)) func(*boundkeypair.JoinStateParams) {
		return func(jsp *boundkeypair.JoinStateParams) {
			token := makeToken(mutators...)
			jsp.Token = &token
		}
	}

	makeInitReq := func(mutators ...func(r *proto.RegisterUsingBoundKeypairInitialRequest)) *proto.RegisterUsingBoundKeypairInitialRequest {
		req := &proto.RegisterUsingBoundKeypairInitialRequest{
			JoinRequest: &types.RegisterUsingTokenRequest{
				HostID:       "host-id",
				Role:         types.RoleBot,
				PublicTLSKey: tlsPublicKey,
				PublicSSHKey: sshPublicKey,
			},
		}
		for _, mutator := range mutators {
			mutator(req)
		}
		return req
	}

	withJoinState := func(signer crypto.Signer, mutators ...func(s *boundkeypair.JoinStateParams)) func(*proto.RegisterUsingBoundKeypairInitialRequest) {
		return func(req *proto.RegisterUsingBoundKeypairInitialRequest) {
			state := makeJoinState(signer, mutators...)
			req.PreviousJoinState = []byte(state)
		}
	}

	makeSolver := func(publicKey string) client.RegisterUsingBoundKeypairChallengeResponseFunc {
		return func(challenge *proto.RegisterUsingBoundKeypairMethodResponse) (*proto.RegisterUsingBoundKeypairMethodRequest, error) {
			switch r := challenge.Response.(type) {
			case *proto.RegisterUsingBoundKeypairMethodResponse_Challenge:
				if r.Challenge.PublicKey != publicKey {
					return nil, trace.BadParameter("wrong public key")
				}

				return &proto.RegisterUsingBoundKeypairMethodRequest{
					Payload: &proto.RegisterUsingBoundKeypairMethodRequest_ChallengeResponse{
						ChallengeResponse: &proto.RegisterUsingBoundKeypairChallengeResponse{
							// For testing purposes, we'll just reply with the
							// public key, to avoid needing to parse the JWT.
							Solution: []byte(publicKey),
						},
					},
				}, nil
			default:
				return nil, trace.BadParameter("invalid response type")
			}
		}
	}

	tests := []struct {
		name string

		token   types.ProvisionTokenV2
		initReq *proto.RegisterUsingBoundKeypairInitialRequest
		solver  client.RegisterUsingBoundKeypairChallengeResponseFunc

		assertError   require.ErrorAssertionFunc
		assertSuccess func(t *testing.T, v2 *types.ProvisionTokenV2, res *client.BoundKeypairRegistrationResponse)
	}{
		{
			// no bound key, no bound bot instance, aka initial join without
			// secret
			name: "initial-join-success",

			token:   makeToken(),
			initReq: makeInitReq(),
			solver:  makeSolver(correctPublicKey),

			assertError: require.NoError,
			assertSuccess: func(t *testing.T, v2 *types.ProvisionTokenV2, _ *client.BoundKeypairRegistrationResponse) {
				// join count should be incremented
				require.Equal(t, uint32(1), v2.Status.BoundKeypair.RecoveryCount)
				require.NotEmpty(t, v2.Status.BoundKeypair.BoundBotInstanceID)
				require.NotEmpty(t, v2.Status.BoundKeypair.BoundPublicKey)
			},
		},
		{
			// no bound key, no bound bot instance, aka initial join without
			// secret
			name: "initial-join-with-wrong-key",

			token:   makeToken(),
			initReq: makeInitReq(),
			solver:  makeSolver(incorrectPublicKey),

			assertError: func(tt require.TestingT, err error, i ...interface{}) {
				require.Error(tt, err)
				require.ErrorContains(tt, err, "wrong public key")
			},
		},
		{
			// bound key, valid bound bot instance, aka "soft join"
			name: "reauth-success",

			token: makeToken(func(v2 *types.ProvisionTokenV2) {
				v2.Status.BoundKeypair.BoundPublicKey = correctPublicKey
				v2.Status.BoundKeypair.BoundBotInstanceID = "asdf"
			}),
			initReq: makeInitReq(func(r *proto.RegisterUsingBoundKeypairInitialRequest) {
				r.JoinRequest.BotInstanceID = "asdf"
			}),
			solver: makeSolver(correctPublicKey),

			assertError: require.NoError,
			assertSuccess: func(t *testing.T, v2 *types.ProvisionTokenV2, _ *client.BoundKeypairRegistrationResponse) {
				// join count should not be incremented
				require.Equal(t, uint32(0), v2.Status.BoundKeypair.RecoveryCount)
			},
		},
		{
			// bound key, seemingly valid bot instance, but wrong key
			// (should be impossible, but should fail anyway)
			name: "reauth-with-wrong-key",

			token: makeToken(func(v2 *types.ProvisionTokenV2) {
				v2.Status.BoundKeypair.BoundPublicKey = correctPublicKey
				v2.Status.BoundKeypair.BoundBotInstanceID = "asdf"
			}),
			initReq: makeInitReq(func(r *proto.RegisterUsingBoundKeypairInitialRequest) {
				r.JoinRequest.BotInstanceID = "asdf"
			}),
			solver: makeSolver(incorrectPublicKey),

			assertError: func(tt require.TestingT, err error, i ...interface{}) {
				require.Error(tt, err)
				require.ErrorContains(tt, err, "wrong public key")
			},
		},
		{
			// bound key but no valid incoming bot instance, i.e. the certs
			// expired and triggered a hard rejoin
			name: "rejoin-success",

			token: makeToken(func(v2 *types.ProvisionTokenV2) {
				v2.Status.BoundKeypair.BoundPublicKey = correctPublicKey
				v2.Status.BoundKeypair.BoundBotInstanceID = "asdf"
			}),
			initReq: makeInitReq(),
			solver:  makeSolver(correctPublicKey),

			assertError: require.NoError,
			assertSuccess: func(t *testing.T, v2 *types.ProvisionTokenV2, _ *client.BoundKeypairRegistrationResponse) {
				require.Equal(t, uint32(1), v2.Status.BoundKeypair.RecoveryCount)

				// Should generate a new bot instance
				require.NotEmpty(t, v2.Status.BoundKeypair.BoundBotInstanceID)
				require.NotEqual(t, "asdf", v2.Status.BoundKeypair.BoundBotInstanceID)
			},
		},
		{
			// Bad state: somehow a key was registered without a bot instance.
			// This should fail and prompt the user to recreate the token.
			name: "bound-key-no-instance",

			token: makeToken(func(v2 *types.ProvisionTokenV2) {
				v2.Status.BoundKeypair.BoundPublicKey = correctPublicKey
			}),
			initReq: makeInitReq(),
			solver:  makeSolver(correctPublicKey),

			assertError: func(tt require.TestingT, err error, i ...interface{}) {
				require.Error(tt, err)
				require.ErrorContains(tt, err, "bad backend state")
			},
		},
		{
			// The client somehow presents certs that refer to a different
			// instance, maybe tried switching auth methods.
			name: "bound-key-wrong-instance",

			token: makeToken(func(v2 *types.ProvisionTokenV2) {
				v2.Status.BoundKeypair.BoundPublicKey = correctPublicKey
				v2.Status.BoundKeypair.BoundBotInstanceID = "qwerty"
			}),
			initReq: makeInitReq(func(r *proto.RegisterUsingBoundKeypairInitialRequest) {
				r.JoinRequest.BotInstanceID = "asdf"
			}),
			solver: makeSolver(correctPublicKey),

			assertError: func(tt require.TestingT, err error, i ...interface{}) {
				require.Error(tt, err)
				require.ErrorContains(tt, err, "bot instance mismatch")
			},
		},
		{
			// TODO: rotation is not yet implemented.
			name: "rotation-requested",

			token: makeToken(func(v2 *types.ProvisionTokenV2) {
				t := time.Now()
				v2.Status.BoundKeypair.BoundPublicKey = correctPublicKey
				v2.Status.BoundKeypair.BoundBotInstanceID = "asdf"
				v2.Spec.BoundKeypair.RotateAfter = &t
				// TODO: test clock?
			}),
			initReq: makeInitReq(),
			solver:  makeSolver(correctPublicKey),

			assertError: func(tt require.TestingT, err error, i ...interface{}) {
				require.Error(tt, err)
				require.ErrorContains(tt, err, "key rotation not yet supported")
			},
		},
		{
			name:        "standard-initial-recovery-success",
			token:       makeToken(withRecovery("standard", 0, 1, "")),
			initReq:     makeInitReq(),
			solver:      makeSolver(correctPublicKey),
			assertError: require.NoError,
			assertSuccess: func(t *testing.T, v2 *types.ProvisionTokenV2, res *client.BoundKeypairRegistrationResponse) {
				require.Equal(t, uint32(1), v2.Status.BoundKeypair.RecoveryCount)

				require.NotNil(t, res)
				require.NotEmpty(t, res.JoinState)
			},
		},
		{
			name:        "standard-success-second-recovery",
			token:       makeToken(withRecovery("standard", 1, 2, "id")),
			initReq:     makeInitReq(withJoinState(jwtSigner, withToken(withRecovery("standard", 1, 2, "id")))),
			solver:      makeSolver(correctPublicKey),
			assertError: require.NoError,
			assertSuccess: func(t *testing.T, v2 *types.ProvisionTokenV2, res *client.BoundKeypairRegistrationResponse) {
				require.Equal(t, uint32(2), v2.Status.BoundKeypair.RecoveryCount)
				require.NotNil(t, res)
				state := parseJoinState(t, res.JoinState)
				require.Equal(t, v2.Status.BoundKeypair.RecoveryCount, state.RecoverySequence)
			},
		},
		{
			name:    "standard-failure-missing-join-state",
			token:   makeToken(withRecovery("standard", 1, 2, "id")),
			initReq: makeInitReq(),
			solver:  makeSolver(correctPublicKey),
			assertError: func(tt require.TestingT, err error, i ...interface{}) {
				require.ErrorContains(tt, err, "previous join state is required")
			},
		},
		{
			name:    "standard-failure-limit-exhausted",
			token:   makeToken(withRecovery("standard", 2, 2, "id")),
			initReq: makeInitReq(withJoinState(jwtSigner, withToken(withRecovery("standard", 2, 2, "id")))),
			solver:  makeSolver(correctPublicKey),
			assertError: func(tt require.TestingT, err error, i ...interface{}) {
				require.ErrorContains(tt, err, "no recovery attempts remaining")
			},
		},
		{
			// Attempts to join with an outdated join state document should fail.
			name:    "standard-failure-recovery-count-mismatch",
			token:   makeToken(withRecovery("standard", 2, 3, "id")),
			initReq: makeInitReq(withJoinState(jwtSigner, withToken(withRecovery("standard", 1, 3, "id")))),
			solver:  makeSolver(correctPublicKey),
			assertError: func(tt require.TestingT, err error, i ...interface{}) {
				require.ErrorContains(tt, err, "join state verification failed")
			},
		},
		{
			name:  "standard-failure-invalid-jwt",
			token: makeToken(withRecovery("standard", 1, 2, "id")),
			initReq: makeInitReq(func(r *proto.RegisterUsingBoundKeypairInitialRequest) {
				r.PreviousJoinState = []byte("asdf")
			}),
			solver: makeSolver(correctPublicKey),
			assertError: func(tt require.TestingT, err error, i ...interface{}) {
				require.ErrorContains(tt, err, "join state verification failed")
			},
		},
		{
			name:    "standard-failure-invalid-jwt-signature",
			token:   makeToken(withRecovery("standard", 1, 2, "id")),
			initReq: makeInitReq(withJoinState(invalidJWTSigner, withToken(withRecovery("standard", 1, 2, "id")))),
			solver:  makeSolver(correctPublicKey),
			assertError: func(tt require.TestingT, err error, i ...interface{}) {
				require.ErrorContains(tt, err, "join state verification failed")
			},
		},
		{
			name:    "standard-failure-invalid-instance-id",
			token:   makeToken(withRecovery("standard", 1, 2, "foo")),
			initReq: makeInitReq(withJoinState(jwtSigner, withToken(withRecovery("standard", 1, 2, "id")))),
			solver:  makeSolver(correctPublicKey),
			assertError: func(tt require.TestingT, err error, i ...interface{}) {
				require.ErrorContains(tt, err, "join state verification failed")
			},
		},
		{
			name:  "standard-failure-invalid-cluster",
			token: makeToken(withRecovery("standard", 1, 2, "foo")),
			initReq: makeInitReq(withJoinState(jwtSigner, withToken(withRecovery("standard", 1, 2, "id")), func(s *boundkeypair.JoinStateParams) {
				s.ClusterName = "wrong-cluster"
			})),
			solver: makeSolver(correctPublicKey),
			assertError: func(tt require.TestingT, err error, i ...interface{}) {
				require.ErrorContains(tt, err, "join state verification failed")
			},
		},
		{
			name:        "relaxed-success-count-over-limit",
			token:       makeToken(withRecovery("relaxed", 1, 0, "id")),
			initReq:     makeInitReq(withJoinState(jwtSigner, withToken(withRecovery("relaxed", 1, 0, "id")))),
			solver:      makeSolver(correctPublicKey),
			assertError: require.NoError,
			assertSuccess: func(t *testing.T, v2 *types.ProvisionTokenV2, res *client.BoundKeypairRegistrationResponse) {
				require.Equal(t, uint32(2), v2.Status.BoundKeypair.RecoveryCount)

				require.NotNil(t, res)
				require.NotEmpty(t, res.JoinState)

				state := parseJoinState(t, res.JoinState)
				require.Equal(t, v2.Status.BoundKeypair.RecoveryCount, state.RecoverySequence)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := types.NewProvisionTokenFromSpecAndStatus(
				tt.name, time.Now().Add(time.Minute), tt.token.Spec, tt.token.Status,
			)
			require.NoError(t, err)
			require.NoError(t, auth.CreateToken(ctx, token))
			tt.initReq.JoinRequest.Token = tt.name

			response, err := auth.RegisterUsingBoundKeypairMethod(ctx, tt.initReq, tt.solver)
			tt.assertError(t, err)

			if tt.assertSuccess != nil {
				pt, err := auth.GetToken(ctx, tt.name)
				require.NoError(t, err)

				ptv2, ok := pt.(*types.ProvisionTokenV2)
				require.True(t, ok)

				tt.assertSuccess(t, ptv2, response)
			}
		})
	}
}
