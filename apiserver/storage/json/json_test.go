/*
Copyright 2015 The Kubernetes Authors.

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

package json_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apiserver/pkg/apis/example"
	examplev1 "k8s.io/apiserver/pkg/apis/example/v1"
	example2v1 "k8s.io/apiserver/pkg/apis/example2/v1"
	"k8s.io/apiserver/pkg/features"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	storagetesting "k8s.io/apiserver/pkg/storage/testing"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/tools/cache"
	featuregatetesting "k8s.io/component-base/featuregate/testing"
	"k8s.io/utils/clock"

	jsonstorage "github.com/grafana/grafana/apiserver/storage/json"
)

var (
	scheme   = runtime.NewScheme()
	codecs   = serializer.NewCodecFactory(scheme)
	errDummy = fmt.Errorf("dummy error")
)

func init() {
	metav1.AddToGroupVersion(scheme, metav1.SchemeGroupVersion)
	utilruntime.Must(example.AddToScheme(scheme))
	utilruntime.Must(examplev1.AddToScheme(scheme))
	utilruntime.Must(example2v1.AddToScheme(scheme))
}

func newPod() runtime.Object     { return &example.Pod{} }
func newPodList() runtime.Object { return &example.PodList{} }

// GetPodAttrs returns labels and fields of a given object for filtering purposes.
func GetPodAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	pod, ok := obj.(*example.Pod)
	if !ok {
		return nil, nil, fmt.Errorf("not a pod")
	}
	return labels.Set(pod.ObjectMeta.Labels), PodToSelectableFields(pod), nil
}

// PodToSelectableFields returns a field set that represents the object
// TODO: fields are not labels, and the validation rules for them do not apply.
func PodToSelectableFields(pod *example.Pod) fields.Set {
	// The purpose of allocation with a given number of elements is to reduce
	// amount of allocations needed to create the fields.Set. If you add any
	// field here or the number of object-meta related fields changes, this should
	// be adjusted.
	podSpecificFieldsSet := make(fields.Set, 5)
	podSpecificFieldsSet["spec.nodeName"] = pod.Spec.NodeName
	podSpecificFieldsSet["spec.restartPolicy"] = string(pod.Spec.RestartPolicy)
	podSpecificFieldsSet["status.phase"] = string(pod.Status.Phase)
	return AddObjectMetaFieldsSet(podSpecificFieldsSet, &pod.ObjectMeta, true)
}

func AddObjectMetaFieldsSet(source fields.Set, objectMeta *metav1.ObjectMeta, hasNamespaceField bool) fields.Set {
	source["metadata.name"] = objectMeta.Name
	if hasNamespaceField {
		source["metadata.namespace"] = objectMeta.Namespace
	}
	return source
}

func checkStorageInvariants(ctx context.Context, t *testing.T, key string) {
	// No-op function since cacher simply passes object creation to the underlying storage.
}

func TestCreate(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestCreate(ctx, t, store, checkStorageInvariants)
}

func TestCreateWithTTL(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestCreateWithTTL(ctx, t, store)
}

func TestCreateWithKeyExist(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestCreateWithKeyExist(ctx, t, store)
}

func TestGet(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestGet(ctx, t, store)
}

func TestUnconditionalDelete(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestUnconditionalDelete(ctx, t, store)
}

func TestConditionalDelete(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestConditionalDelete(ctx, t, store)
}

func TestDeleteWithSuggestion(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestDeleteWithSuggestion(ctx, t, store)
}

func TestDeleteWithSuggestionAndConflict(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestDeleteWithSuggestionAndConflict(ctx, t, store)
}

func TestDeleteWithSuggestionOfDeletedObject(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestDeleteWithSuggestionOfDeletedObject(ctx, t, store)
}

func TestValidateDeletionWithSuggestion(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestValidateDeletionWithSuggestion(ctx, t, store)
}

func TestPreconditionalDeleteWithSuggestion(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestPreconditionalDeleteWithSuggestion(ctx, t, store)
}

func TestList(t *testing.T) {
	ctx, store := testSetupWithJSONStorage(t)
	storagetesting.RunTestList(ctx, t, store, compactStorage(), true)
}

func TestListWithListFromCache(t *testing.T) {
	defer featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.ConsistentListFromCache, true)()
	ctx, store := testSetupWithJSONStorage(t)
	storagetesting.RunTestList(ctx, t, store, compactStorage(), true)
}

func TestListWithoutPaging(t *testing.T) {
	ctx, store := testSetup(t, withoutPaging)
	storagetesting.RunTestListWithoutPaging(ctx, t, store)
}

func TestGetListNonRecursive(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestGetListNonRecursive(ctx, t, store)
}

func checkStorageCalls(t *testing.T, pageSize, estimatedProcessedObjects uint64) {
	// No-op function for now, since store passes pagination calls to underlying storage.
}

func TestListContinuation(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestListContinuation(ctx, t, store, checkStorageCalls)
}

func TestListPaginationRareObject(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestListPaginationRareObject(ctx, t, store, checkStorageCalls)
}

func TestListContinuationWithFilter(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestListContinuationWithFilter(ctx, t, store, checkStorageCalls)
}

func TestListInconsistentContinuation(t *testing.T) {
	ctx, store := testSetup(t)

	// TODO(#109831): Enable use of this by setting compaction.
	storagetesting.RunTestListInconsistentContinuation(ctx, t, store, nil)
}

func TestConsistentList(t *testing.T) {
	// TODO(#109831): Enable use of this test and run it.
}

func TestGuaranteedUpdate(t *testing.T) {
	// TODO(#109831): Enable use of this test and run it.
}

func TestGuaranteedUpdateWithTTL(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestGuaranteedUpdateWithTTL(ctx, t, store)
}

func TestGuaranteedUpdateChecksStoredData(t *testing.T) {
	// TODO(#109831): Enable use of this test and run it.
}

func TestGuaranteedUpdateWithConflict(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestGuaranteedUpdateWithConflict(ctx, t, store)
}

func TestGuaranteedUpdateWithSuggestionAndConflict(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestGuaranteedUpdateWithSuggestionAndConflict(ctx, t, store)
}

func TestTransformationFailure(t *testing.T) {
	// TODO(#109831): Enable use of this test and run it.
}

func TestCount(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestCount(ctx, t, store)
}

func TestWatch(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestWatch(ctx, t, store)
}

func TestWatchFromZero(t *testing.T) {
	ctx, store := testSetupWithJSONStorage(t)
	storagetesting.RunTestWatchFromZero(ctx, t, store, compactStorage())
}

func TestDeleteTriggerWatch(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestDeleteTriggerWatch(ctx, t, store)
}

func TestWatchFromNonZero(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestWatchFromNonZero(ctx, t, store)
}

func TestDelayedWatchDelivery(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestDelayedWatchDelivery(ctx, t, store)
}

func TestWatchError(t *testing.T) {
	// TODO(#109831): Enable use of this test and run it.
}

func TestWatchContextCancel(t *testing.T) {
	// TODO(#109831): Enable use of this test and run it.
}

func TestWatcherTimeout(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestWatcherTimeout(ctx, t, store)
}

func TestWatchDeleteEventObjectHaveLatestRV(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestWatchDeleteEventObjectHaveLatestRV(ctx, t, store)
}

func TestWatchInitializationSignal(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestWatchInitializationSignal(ctx, t, store)
}

func TestClusterScopedWatch(t *testing.T) {
	ctx, store := testSetup(t, withClusterScopedKeyFunc, withSpecNodeNameIndexerFuncs)
	storagetesting.RunTestClusterScopedWatch(ctx, t, store)
}

func TestNamespaceScopedWatch(t *testing.T) {
	ctx, store := testSetup(t, withSpecNodeNameIndexerFuncs)
	storagetesting.RunTestNamespaceScopedWatch(ctx, t, store)
}

func TestWatchDispatchBookmarkEvents(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestWatchDispatchBookmarkEvents(ctx, t, store)
}

func TestWatchBookmarksWithCorrectResourceVersion(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunTestOptionalWatchBookmarksWithCorrectResourceVersion(ctx, t, store)
}

func TestSendInitialEventsBackwardCompatibility(t *testing.T) {
	ctx, store := testSetup(t)
	storagetesting.RunSendInitialEventsBackwardCompatibility(ctx, t, store)
}

// ===================================================
// Test-setup related function are following.
// ===================================================

type tearDownFunc func()

type setupOptions struct {
	resourcePrefix string
	keyFunc        func(runtime.Object) (string, error)
	indexerFuncs   map[string]storage.IndexerFunc
	pagingEnabled  bool
	clock          clock.WithTicker
}

type setupOption func(*setupOptions)

func withDefaults(options *setupOptions) {
	tmpDir, err := os.MkdirTemp("/tmp", "grafana-apiserver-*")
	if err != nil {
		panic(fmt.Sprintf("unexpected error: %s", err.Error()))
	}
	prefix := path.Join(tmpDir, "pods")

	options.resourcePrefix = prefix
	options.keyFunc = func(obj runtime.Object) (string, error) { return storage.NamespaceKeyFunc(prefix, obj) }
	options.pagingEnabled = true
	options.clock = clock.RealClock{}
}

func withClusterScopedKeyFunc(options *setupOptions) {
	options.keyFunc = func(obj runtime.Object) (string, error) {
		return storage.NoNamespaceKeyFunc(options.resourcePrefix, obj)
	}
}

func withSpecNodeNameIndexerFuncs(options *setupOptions) {
	options.indexerFuncs = map[string]storage.IndexerFunc{
		"spec.nodeName": func(obj runtime.Object) string {
			pod, ok := obj.(*example.Pod)
			if !ok {
				return ""
			}
			return pod.Spec.NodeName
		},
	}
}

func withoutPaging(options *setupOptions) {
	options.pagingEnabled = false
}

func testSetup(t *testing.T, opts ...setupOption) (context.Context, storage.Interface) {
	ctx, store := testSetupWithJSONStorage(t, opts...)
	return ctx, store
}

func testSetupWithJSONStorage(t *testing.T, opts ...setupOption) (context.Context, storage.Interface) {
	setupOpts := setupOptions{}
	opts = append([]setupOption{withDefaults}, opts...)
	for _, opt := range opts {
		opt(&setupOpts)
	}

	var cacher *cache.Indexers
	store, _, err := jsonstorage.NewStorage(
		&storagebackend.ConfigForResource{
			Config:        storagebackend.Config{Type: "json", Prefix: setupOpts.resourcePrefix, Paging: setupOpts.pagingEnabled, Codec: codecs.LegacyCodec(examplev1.SchemeGroupVersion)},
			GroupResource: examplev1.Resource("pods"),
		},
		setupOpts.resourcePrefix,
		setupOpts.keyFunc,
		newPod,
		newPodList,
		GetPodAttrs,
		setupOpts.indexerFuncs,
		cacher,
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	// Inject one list error to make sure we test the relist case.
	wrappedStorage := &storagetesting.StorageInjectingListErrors{
		Interface: store,
		Errors:    1,
	}

	ctx := context.Background()

	// Since some tests depend on the fact that GetList shouldn't fail,
	// we wait until the error from the underlying storage is consumed.
	//if err := wait.PollInfinite(100*time.Millisecond, wrappedStorage.ErrorsConsumed); err != nil {
	//	t.Fatalf("Failed to inject list errors: %v", err)
	//}

	return ctx, wrappedStorage
}

func compactStorage() storagetesting.Compaction {
	return func(ctx context.Context, t *testing.T, resourceVersion string) {
	}
}
