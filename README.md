# gmail-util

A CLI tool to export and purge Gmail messages. It downloads emails to a [MBox file format](https://en.wikipedia.org/wiki/Mbox) that can later be opened using desktop mail client tools like Thunderbird.

# Usage

## Install

Download the executable `gmail-util` from the latest [release](https://github.com/rinconjc/gmail-util/releases) for your platform and OS, and follow the below instructions to configure and run.

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
    
    

## Usage Examples

* Print help

```shell
./gmail-util -h

# Usage:
#         ./gmail-util <command> [arguments]
# 
# The commands are:
# 
#         export  Exports messages to a file
#         purge   Deletes messages from Gmail

```

* Export all messages received in 2010

```shell
./gmail-util export -q 'to:me after:2009 before:2011' -o emails2010 -a -c 6

# Exporting messages matching: to:me after:2009 before:2011 to file: emails2010
# Messages exported: 1489
# Done

```

* Purge all messages from ebay@ebay.com.au

```shell
./gmail-util purge -q 'from:ebay@ebay.com.au' -a

# Deleting messages matching: from:ebay@ebay.com.au
# Messages deleted:115
# Done
```

* Purge all unread messages received before 2020 (For me old unread emails are mostly spam)

```shell
./gmail-util purge -q 'is:unread before:2020' -a

```




