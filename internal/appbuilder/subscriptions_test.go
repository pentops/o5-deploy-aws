package appbuilder

import (
	"encoding/json"
	"testing"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"google.golang.org/protobuf/proto"
)

func TestEventBusRules(t *testing.T) {
	var (
		testAppName = ""

		localEnvRef = cloudformation.Ref(EnvNameParameter)

		fullNameRef = cloudformation.Join("/", []string{
			cloudformation.Ref(EnvNameParameter),
			testAppName,
		})
	)

	for _, tc := range []struct {
		name  string
		want  []map[string]interface{}
		input *application_pb.Runtime
	}{
		{
			name:  "no rules",
			want:  nil,
			input: &application_pb.Runtime{},
		},
		{
			name: "one topic rule",
			input: &application_pb.Runtime{
				Subscriptions: []*application_pb.Subscription{{
					Name: "test",
				}},
			},
			want: []map[string]interface{}{{
				"detail": map[string]interface{}{
					"sourceEnv":        []interface{}{localEnvRef},
					"destinationTopic": []interface{}{"test"},
					"reply": map[string]interface{}{
						"replyTo": []interface{}{
							map[string]interface{}{
								"exists": false,
							},
							fullNameRef,
						},
					},
				},
			}},
		},
		{
			name: "wildcard env",
			input: &application_pb.Runtime{
				Subscriptions: []*application_pb.Subscription{{
					Name:    "test",
					EnvName: proto.String("*"),
				}},
			},
			want: []map[string]interface{}{{
				"detail": map[string]interface{}{
					"destinationTopic": []interface{}{"test"},
					"reply": map[string]interface{}{
						"replyTo": []interface{}{
							map[string]interface{}{
								"exists": false,
							},
							fullNameRef,
						},
					},
				},
			}},
		},
		{
			name: "two topic rules",
			input: &application_pb.Runtime{
				Subscriptions: []*application_pb.Subscription{{
					Name: "test",
				}, {
					Name: "test2",
				}},
			},
			want: []map[string]interface{}{{
				"detail": map[string]interface{}{
					"sourceEnv":        []interface{}{localEnvRef},
					"destinationTopic": []interface{}{"test", "test2"},
					"reply": map[string]interface{}{
						"replyTo": []interface{}{
							map[string]interface{}{
								"exists": false,
							},
							fullNameRef,
						},
					},
				},
			}},
		},
		{
			name: "one service rule",
			input: &application_pb.Runtime{
				Subscriptions: []*application_pb.Subscription{{
					Name: "/foo.v1.Bar",
				}},
			},
			want: []map[string]interface{}{{
				"detail": map[string]interface{}{
					"sourceEnv":   []interface{}{localEnvRef},
					"grpcService": []interface{}{"foo.v1.Bar"},
					"reply": map[string]interface{}{
						"replyTo": []interface{}{
							map[string]interface{}{
								"exists": false,
							},
							fullNameRef,
						},
					},
				},
			}},
		},
		{
			name: "one method rule",
			input: &application_pb.Runtime{
				Subscriptions: []*application_pb.Subscription{{
					Name: "/foo.v1.Bar/Baz",
				}},
			},
			want: []map[string]interface{}{{
				"detail": map[string]interface{}{
					"sourceEnv":   []interface{}{localEnvRef},
					"grpcService": []interface{}{"foo.v1.Bar"},
					"grpcMethod":  []interface{}{"Baz"},
					"reply": map[string]interface{}{
						"replyTo": []interface{}{
							map[string]interface{}{
								"exists": false,
							},
							fullNameRef,
						},
					},
				},
			}},
		},
		{
			name: "group methods by service",
			input: &application_pb.Runtime{
				Subscriptions: []*application_pb.Subscription{{
					Name: "/foo.v1.Bar/Baz",
				}, {
					Name: "/foo.v1.Bar/Qux",
				}},
			},
			want: []map[string]interface{}{{
				"detail": map[string]interface{}{
					"sourceEnv":   []interface{}{localEnvRef},
					"grpcService": []interface{}{"foo.v1.Bar"},
					"grpcMethod":  []interface{}{"Baz", "Qux"},
					"reply": map[string]interface{}{
						"replyTo": []interface{}{
							map[string]interface{}{
								"exists": false,
							},
							fullNameRef,
						},
					},
				},
			}},
		},
		{
			name: "one service and method rule",
			input: &application_pb.Runtime{
				Subscriptions: []*application_pb.Subscription{{
					Name: "/foo.v1.Bar/Baz",
				}, {
					Name: "/foo.v1.Qux",
				}},
			},
			want: []map[string]interface{}{{
				"detail": map[string]interface{}{
					"$or": []interface{}{
						map[string]interface{}{
							"sourceEnv":   []interface{}{localEnvRef},
							"grpcService": []interface{}{"foo.v1.Qux"},
							"reply": map[string]interface{}{
								"replyTo": []interface{}{
									map[string]interface{}{
										"exists": false,
									},
									fullNameRef,
								},
							},
						},
						map[string]interface{}{
							"sourceEnv":   []interface{}{localEnvRef},
							"grpcService": []interface{}{"foo.v1.Bar"},
							"grpcMethod":  []interface{}{"Baz"},
							"reply": map[string]interface{}{
								"replyTo": []interface{}{
									map[string]interface{}{
										"exists": false,
									},
									fullNameRef,
								},
							},
						},
					},
				},
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.input.Containers) == 0 {
				tc.input.Containers = []*application_pb.Container{{
					Name: "default",
				}}
			}

			got, err := buildSubscriptionPlan(testAppName, tc.input)
			if err != nil {
				t.Fatal(err)
			}

			if len(got.eventBusSubscriptions) != len(tc.want) {
				t.Errorf("expected %d rules, got %d", len(tc.want), len(got.eventBusSubscriptions))
			}

			for idx := range tc.want {
				if idx >= len(got.eventBusSubscriptions) {
					t.Errorf("extra rule %d rules, got %d", len(tc.want), len(got.eventBusSubscriptions))
					continue
				}

				gotAsJSON, err := json.Marshal(got.eventBusSubscriptions[idx].eventPattern)
				if err != nil {
					t.Fatal(err)
				}

				gotAsMap := map[string]interface{}{}
				if err := json.Unmarshal(gotAsJSON, &gotAsMap); err != nil {
					t.Fatal(err)
				}

				t.Log(string(gotAsJSON))
				assertJSONMapEqual(t, []string{"$"}, tc.want[idx], gotAsMap)
			}
		})
	}

}
