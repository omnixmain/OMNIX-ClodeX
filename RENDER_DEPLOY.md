# How to Deploy to Render.com (Free)

This guide will help you deploy your TG-FileStreamBot to Render.com using the new configuration.

## Step 1: Push Changes to GitHub
Since we updated the `Dockerfile` and added `render.yaml`, you need to push these changes to your GitHub repository first.
(I assume you know how to git push, or you can ask me to help with that).

## Step 2: Create Account & Project
1.  Go to [dashboard.render.com](https://dashboard.render.com/).
2.  Login with **GitHub**.
3.  Click **New +** and select **Blueprints**.
4.  Connect your GitHub repository (`TG-FileStreamBot`).
5.  Render will automatically detect the `render.yaml` file.

## Step 3: Configure Environment
You will see a form to enter your Environment Variables. Enter the following:

| Key | Value |
| :--- | :--- |
| `API_ID` | Your Telegram API ID |
| `API_HASH` | Your Telegram API Hash |
| `BOT_TOKEN` | Your Bot Token |
| `LOG_CHANNEL` | Channel ID for logs (e.g. -100xxxx) |
| `HOST` | Leave empty for now, we will set this after deploy |

*   **PORT** is already set to 8080 by the file.
*   Click **Apply**.

## Step 4: Finalize Setup
1.  Wait for the deployment to finish. It might take 2-3 minutes.
2.  Once deployed, you will see a URL like `https://omnixclodex-xxxx.onrender.com`.
3.  **Copy this URL**.
4.  Go to the **Environment** tab in your Render dashboard for this service.
5.  Add/Edit the `HOST` variable:
    *   **Key**: `HOST`
    *   **Value**: `https://omnixclodex-xxxx.onrender.com` (Your actual URL)
6.  Save Changes. The bot will restart automatically.

## Step 5: Keep it Alive (UptimeRobot)
Render Free tier spins down after 15 minutes of inactivity. To prevent this:
1.  Go to [uptimerobot.com](https://uptimerobot.com/).
2.  Create a **New Monitor**.
3.  **Monitor Type**: HTTP(s).
4.  **Friendly Name**: My Bot.
5.  **URL**: `https://omnixclodex-xxxx.onrender.com` (Your Render URL).
6.  **Monitoring Interval**: 5 minutes.
7.  Create Monitor.

Now your bot will stay online 24/7!
