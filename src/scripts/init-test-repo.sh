#!/bin/sh
# Initialize a test Git repository

# Create test repo directory if it doesn't exist
REPOS_PATH="$(pwd)/volumes/repos"
TEST_REPO_NAME="test-repo.git"
TEST_REPO_PATH="$REPOS_PATH/$TEST_REPO_NAME"
TMP_DIR=$(mktemp -d)
mkdir -p $REPOS_PATH

rm -fr $TEST_REPO_PATH
# Initialize test repository if not exists
if [ ! -d $TEST_REPO_PATH ]; then
  mkdir -p $REPOS_PATH
  cd $REPOS_PATH
  git init --bare $TEST_REPO_NAME
  chmod -R 777 $TEST_REPO_NAME
  cd $TEST_REPO_NAME
  git config http.receivepack true
  mkdir -p $TMP_DIR
  cd $TMP_DIR
  git init
  echo "hello" > README.md
  git add README.md
  git config user.name "nginx"
  git config user.email "nginx@example.com"
  git commit -m "Initial commit"
  git remote add origin $TEST_REPO_PATH
  git push origin master
fi

echo "Test repository initialized at $BARE_REPO_PATH"
echo "You can clone it with: git clone http://localhost:8080/$TEST_REPO_NAME"
