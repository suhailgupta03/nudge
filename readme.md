**Design Overview**
The Nudge system consists of three main components: A machine learning-based effort estimation
model that predicts the lifetime of a given pull request, an activity detection module to establish
what the current state of the pull request is, and an actor determination module to identify who
would be need to take action

* _Prediction Model._ The Nudge system leverages a prediction model to determine the lifetime for
  every pull request. The model is a linear regression model. We performed
  the regression analysis to understand the weights of each of the features and how they impact the
  ability of the model to accurately predict the lifetime for a given pull request. We use historical
  pull request data to extract some of the features and the dependant variable (pull request lifetime).
  For the repositories where we have enough training data, i.e., at least thousands of data points
  (or pull requests), we train a repository-specific model. If the repository is small or new and it does
  not have many pull requests that is completed, then we use a global model that is trained on all the
  repositories’ data. Once the repository matures and records enough activity, we train a repositoryspecific model and deploy it. The models are retrained, through an offline process, periodically, to
  adjust to the changes in the feature weights and changing repository dynamics. Every time the
  model is retrained, we use a moving window to fetch the data from the past 2 years (from the date
  of retraining) to make sure the training data reflect the ever-changing dynamics and takes into
  account the changes happening to the development processes.
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