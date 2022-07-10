# gmail-util
A CLI tool to export and purge Gmail messages

# Usage

## Configure OAuth2 Client

* Create a new project in your Google account from https://console.cloud.google.com/projectcreate. If you already have a project skip this step

* Enable Gmail API for your project above: https://console.cloud.google.com/projectselector2/apis/library/gmail.googleapis.com?organizationId=0 . Make sure you select a project from the given choices, and click Enable in the next screen.

    * Create a new credential:
      + From the API/Service details above, select the **CREDENTIALS** tab click on **+CREATE CREDENTIALS** link.
      + Select the first option (**OAuth client ID**).
      + Select **Desktop app** and enter any name
      + Then click **CREATE**.
      + From the pop-up window select the **DOWNLOAD JSON** link and click **Ok**
      + Copy the file downloaded in the previous step to your **$HOME/.config/gmail-secret.json**
    
    
## Install

Download the command from Github releases, or use brew:

```shell
brew install gmail-util
```

## Running the command

```shell
gmail-util -h

```


