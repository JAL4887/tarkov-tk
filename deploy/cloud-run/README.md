# Cloud Run worker-pool deployment

This deployment runs the Discord bot as a single continuously running Cloud Run worker-pool instance.

## Deployment shape

- Cloud Run worker pool: `tarkov-tk`
- Instances: `1`
- CPU: `0.08`
- Memory: `512Mi`
- Runtime identity: dedicated user-managed Google service account
- Firestore authentication: Application Default Credentials from the attached service account
- Bot configuration: one Secret Manager secret mounted as `/secrets/tarkov-tk/config.toml`

The production configuration intentionally leaves `firebase.serviceAccountFilePath` empty. On Google Cloud the Firebase Admin SDK uses the service account attached to the worker pool, so no service-account JSON key is copied into the container.

## 1. Prepare the production config

Copy `deploy/cloud-run/config.example.toml` to a temporary local file and replace the placeholders. Never commit the completed file.

The production file should contain the Discord bot token, guild ID, Firebase project ID, and:

```toml
[firebase]
projectId = "YOUR_PROJECT_ID"
serviceAccountFilePath = ""
```

Keep `removeCommands = false` so normal restarts do not remove the Discord slash commands.

## 2. Set Cloud Shell variables

Run these commands from Google Cloud Shell after selecting the Firebase/Google Cloud project used by the bot:

```bash
export PROJECT_ID="YOUR_PROJECT_ID"
export REGION="us-east1"
export WORKER_POOL="tarkov-tk"
export RUNTIME_SA="tarkov-tk-runtime"
export CONFIG_SECRET="tarkov-tk-config"
export RUNTIME_SA_EMAIL="${RUNTIME_SA}@${PROJECT_ID}.iam.gserviceaccount.com"

gcloud config set project "$PROJECT_ID"
```

## 3. Enable required APIs

```bash
gcloud services enable \
  run.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  secretmanager.googleapis.com \
  firestore.googleapis.com
```

## 4. Create the runtime service account

```bash
gcloud iam service-accounts describe "$RUNTIME_SA_EMAIL" >/dev/null 2>&1 || \
  gcloud iam service-accounts create "$RUNTIME_SA" \
    --display-name="Tarkov TK bot runtime"
```

Grant only the Firestore data access needed by the bot:

```bash
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${RUNTIME_SA_EMAIL}" \
  --role="roles/datastore.user"
```

## 5. Create the configuration secret

Create a Secret Manager secret named `tarkov-tk-config` whose value is the completed production TOML from step 1.

After the secret exists, grant the runtime service account permission to read it:

```bash
gcloud secrets add-iam-policy-binding "$CONFIG_SECRET" \
  --member="serviceAccount:${RUNTIME_SA_EMAIL}" \
  --role="roles/secretmanager.secretAccessor"
```

## 6. Get the source in Cloud Shell

```bash
git clone https://github.com/JAL4887/tarkov-tk.git
cd tarkov-tk
git fetch origin
git checkout feature/cloud-run-deployment
```

For later deployments, use the branch or commit that has already passed local build/test verification.

## 7. Deploy the worker pool

The repository Dockerfile is used automatically by source deployment.

```bash
gcloud run worker-pools deploy "$WORKER_POOL" \
  --source . \
  --region "$REGION" \
  --instances 1 \
  --cpu 0.08 \
  --memory 512Mi \
  --service-account "$RUNTIME_SA_EMAIL" \
  --set-secrets "/secrets/tarkov-tk/config.toml=${CONFIG_SECRET}:latest" \
  --args="--config,/secrets/tarkov-tk/config.toml,serve"
```

The worker pool must remain at one instance for the Discord bot to remain connected continuously.

## 8. Verify the deployment

In Google Cloud Console, open Cloud Run -> Worker pools -> `tarkov-tk` -> Logs.

Look for the normal startup messages indicating that the bot logged into Discord, registered the guild commands, and is waiting for shutdown.

Then verify in Discord that:

- the `tarkov-tk` bot is online;
- `/tkstats` works;
- `/disappointmentstats` works;
- `/stats` works;
- a normal worker-pool revision/restart does not remove commands.

## Updating the deployed bot

After a new feature is merged and verified, update the Cloud Shell checkout and redeploy from source:

```bash
git checkout master
git pull --ff-only origin master

gcloud run worker-pools deploy "$WORKER_POOL" \
  --source . \
  --region "$REGION" \
  --instances 1 \
  --cpu 0.08 \
  --memory 512Mi \
  --service-account "$RUNTIME_SA_EMAIL" \
  --set-secrets "/secrets/tarkov-tk/config.toml=${CONFIG_SECRET}:latest" \
  --args="--config,/secrets/tarkov-tk/config.toml,serve"
```

Do not put `service-account-file.json`, the Discord bot token, or a completed production TOML file into the container image or Git repository.
