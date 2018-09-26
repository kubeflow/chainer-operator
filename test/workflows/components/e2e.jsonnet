local k = import 'k.libsonnet';
local util = import 'util.libsonnet';

// default parameters.
local defaultParams = {
  project:: "kubeflow-ci",
  zone:: "us-east1-d",
  // Default registry to use.
  registry:: "gcr.io/" + defaultParams.project,

  // The image tag to use.
  // Defaults to a value based on the name.
  versionTag:: null,

  // The name of the secret containing GCP credentials.
  gcpCredentialsSecretName:: "kubeflow-testing-credentials",
};

local params = defaultParams + std.extVar("__ksonnet/params").components.e2e;
local namespace = params.namespace;
local name = params.name;
local prowEnv = util.parseEnv(params.prow_env);
local bucket = params.bucket;

local workflow = {
  // mountPath is the directory where the volume to store the test data
  // should be mounted.
  local mountPath = "/mnt/" + "test-data-volume",
  // testDir is the root directory for all data for a particular test run.
  local testDir = mountPath + "/" + name,
  // outputDir is the directory to sync to GCS to contain the output for this job.
  local outputDir = testDir + "/output",
  local artifactsDir = outputDir + "/artifacts",
  local goDir = testDir + "/go",
  // Source directory where all repos should be checked out
  local srcRootDir = testDir + "/src",
  // The directory containing the kubeflow/chainer-operator repo
  local srcDir = srcRootDir + "/kubeflow/chainer-operator",
  local testWorkerImage = "gcr.io/kubeflow-ci/test-worker",
  // The name of the NFS volume claim to use for test files.
  // local nfsVolumeClaim = "kubeflow-testing";
  local nfsVolumeClaim = "nfs-external",
  // The name to use for the volume to use to contain test data.
  local dataVolume = "kubeflow-test-volume",
  local versionTag = if params.versionTag != null then
    params.versionTag
    else name,
  local chainerJobImage = params.registry + "/chainer-operator:" + versionTag,

  // The namespace on the cluster we spin up to deploy into.
  local deployNamespace = "kubeflow",
  // The directory within the kubeflow_testing submodule containing
  // py scripts to use.
  local k8sPy = srcDir,
  local kubeflowPy = srcRootDir + "/kubeflow/testing/py",
  local kfctlDir = srcRootDir + "/kubeflow/kubeflow",

  local project = params.project,
  // GKE cluster to use
  // We need to truncate the cluster to no more than 40 characters because
  // cluster names can be a max of 40 characters.
  // We expect the suffix of the cluster name to be unique salt.
  // We prepend a z because cluster name must start with an alphanumeric character
  // and if we cut the prefix we might end up starting with "-" or other invalid
  // character for first character.
  local cluster =
    if std.length(name) > 20 then
      "z" + std.substr(name, std.length(name) - 19, 19)
    else
      name,
  local zone = params.zone,
  local registry = params.registry,
  local chart = srcDir + "/chainer-operator-chart",

  // Build an Argo template to execute a particular command.
  // step_name: Name for the template
  // command: List to pass as the container command.
  buildTemplate(step_name, image, command) :: {
    name: step_name,
    container: {
      command: command,
      image: image,
      workingDir: srcDir,
      env: [
        {
          // Add the source directories to the python path.
          name: "PYTHONPATH",
          value: k8sPy + ":" + kubeflowPy,
        },
        {
          // Set the GOPATH
          name: "GOPATH",
          value: goDir,
        },
        {
          name: "KFCTL_DIR",
          value: kfctlDir,
        },
        {
          name: "CLUSTER_NAME",
          value: cluster,
        },
        {
          name: "GCP_ZONE",
          value: zone,
        },
        {
          name: "GCP_PROJECT",
          value: project,
        },
        {
          name: "GCP_REGISTRY",
          value: registry,
        },
        {
          name: "DEPLOY_NAMESPACE",
          value: deployNamespace,
        },
        {
          name: "GOOGLE_APPLICATION_CREDENTIALS",
          value: "/secret/gcp-credentials/key.json",
        },
        {
          name: "GIT_TOKEN",
          valueFrom: {
            secretKeyRef: {
              name: "github-token",
              key: "github_token",
            },
          },
        },
      ] + prowEnv,
      volumeMounts: [
        {
          name: dataVolume,
          mountPath: mountPath,
        },
        {
          name: "github-token",
          mountPath: "/secret/github-token",
        },
        {
          name: "gcp-credentials",
          mountPath: "/secret/gcp-credentials",
        },
      ],
    },
  },  // buildTemplate

  apiVersion: "argoproj.io/v1alpha1",
  kind: "Workflow",
  metadata: {
    name: name,
    namespace: namespace,
  },
  // TODO(jlewi): Use OnExit to run cleanup steps.
  spec: {
    entrypoint: "e2e",
    volumes: [
      {
        name: "github-token",
        secret: {
          secretName: "github-token",
        },
      },
      {
        name: "gcp-credentials",
        secret: {
          secretName: params.gcpCredentialsSecretName,
        },
      },
      {
        name: dataVolume,
        persistentVolumeClaim: {
          claimName: nfsVolumeClaim,
        },
      },
    ],  // volumes
    // onExit specifies the template that should always run when the workflow completes.          
    onExit: "exit-handler",
    templates: [
      {
        name: "e2e",
        steps: [
          [{
            name: "checkout",
            template: "checkout",
          }],
          [
            {
              name: "build",
              template: "build",
            },
            {
              name: "create-pr-symlink",
              template: "create-pr-symlink",
            },
          ],
          [  // Setup cluster needs to run after build because we depend on the chart
            // created by the build statement.
            {
              name: "setup-cluster",
              template: "setup-cluster",
            },
          ],
          [
            {
              name: "run-tests",
              template: "run-tests",
            },
          ],
        ],
      },
      {
        name: "exit-handler",
        steps: [
          [{
            name: "teardown-cluster",
            template: "teardown-cluster",

          }],
          [{
            name: "copy-artifacts",
            template: "copy-artifacts",
          }],
        ],
      },
      {
        name: "checkout",
        container: {
          command: [
            "/usr/local/bin/checkout.sh",
            srcRootDir,
          ],
          env: prowEnv + [{
            name: "EXTRA_REPOS",
            value: "kubeflow/kubeflow@HEAD;kubeflow/testing@HEAD",
          }],
          image: testWorkerImage,
          volumeMounts: [
            {
              name: dataVolume,
              mountPath: mountPath,
            },
          ],
        },
      },  // checkout
      workflow.buildTemplate("setup-cluster",testWorkerImage, [
        "scripts/create-cluster.sh",
      ]),  // setup cluster
      workflow.buildTemplate("run-tests", testWorkerImage, [
        "scripts/run-test.sh",
      ]),  // run tests
      workflow.buildTemplate("create-pr-symlink", testWorkerImage, [
        "python",
        "-m",
        "kubeflow.testing.prow_artifacts",
        "--artifacts_dir=" + outputDir,
        "create_pr_symlink",
        "--bucket=" + bucket,
      ]),  // create-pr-symlink
      workflow.buildTemplate("teardown-cluster",testWorkerImage, [
        "scripts/delete-cluster.sh",
        ]),  // teardown cluster
      workflow.buildTemplate("copy-artifacts", testWorkerImage, [
        "python",
        "-m",
        "kubeflow.testing.prow_artifacts",
        "--artifacts_dir=" + outputDir,
        "copy_artifacts",
        "--bucket=" + bucket,
      ]),  // copy-artifacts
      workflow.buildTemplate("build", testWorkerImage, [
        "scripts/build.sh",
      ]),  // build
    ],  // templates
  },
};

local release = import "kubeflow/automation/release.libsonnet";

std.prune(k.core.v1.list.new([
  workflow,
  release.parts(params.namespace, std.strReplace(params.name, "e2e", "rl"), overrides=params).release
]))
