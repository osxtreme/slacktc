# slacktc
Slack TC tools

Slack slash commands, deployable with Up to Lambda:
slacktc-time - link to /time, default to time a website
slacktc-quote - link to /quote - look up stock quotes with Yahoo Finance CSV API

To deploy with Up:
  Demo tutorial: https://medium.freecodecamp.org/creating-serverless-slack-commands-in-minutes-with-up-f04ce0cfd52c
  
  Save your channeltoken to a file, e.g. channeltoken

  Make an up.json to reference your aws token and app info:
{
  "name": "slacktc-quote",
  "profile": "slacktc",
  "environment": {
    "SLACK_APP_VERIFY_TOKEN": "token goes here"
  }   
}
