**Start Dev Server**
```shell
make dev-docker
```

**Create a release**
```shell
make release
```


**Create build**
```shell
make docker
```

**Design Overview**
The Nudge system consists of three main components: A machine learning-based effort estimation
model that predicts the lifetime of a given pull request, an activity detection module to establish
what the current state of the pull request is, and an actor determination module to identify who
would be need to take action

* _Prediction Model._ As of today this has not been open sourced and will be be made available soon.
* _Activity Detection_ The role of the activity detection module is to help the Nudge system understand if there has been any activity performed by the author or the reviewer of the pull request of
  late. This helps the Nudge system not send a notification, even though the lifetime of the pull request has exceeded its predicted lifetime. This module serves as a gatekeeper that gives the Nudge
  system a “go” or “no go” by observing various signals in the pull request environment.
* _Actor Identification_. The primary goal of this module is to determine the blocker of the change
  (the author or a reviewer) and engage them in the notification, by explicitly mentioning them. This
  module comes into action once the pull request meets the criteria set by the prediction module
  and the Activity Detection modules. Once the Nudge system is ready to send the notification, the
  Actor Identification module provides information to the Nudge notification system to direct the
  notification toward the change blocker.

![workflow](data/flow.png)

_Nudge Workflow._ The three modules are combined with a notification system to form Nudge as
shown in Figure above.