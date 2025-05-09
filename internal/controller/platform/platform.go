/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package platform

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/NVIDIA/k8s-nim-operator/internal/shared"
)

// Platform defines the methods required for an inference platform integration.
type Platform interface {
	Delete(ctx context.Context, r shared.Reconciler, resource client.Object) error
	Sync(ctx context.Context, r shared.Reconciler, resource client.Object) (ctrl.Result, error)
}
