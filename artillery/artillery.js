function randomPick(requestParams, response, context, ee, next) {
  const x = JSON.parse(response.body);
  const randQ = getRandomInt(0, x.QuestionsAndAnswers.length - 1);
  const randA = getRandomInt(0, x.QuestionsAndAnswers[randQ].Answers.length - 1);
  context.vars['questionId'] = randQ;
  context.vars['answerId'] = randA;
  return next();
}

function getRandomInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

module.exports = {
  randomPick: randomPick
}
