import {Answers} from './Answers';

export class QuestionsAndAnswers {
  public Question: string;
  public QuestionId: number;
  public Votes: number[];
  public VoteLabels: number[];
  public Answers: Answers[];
  public Voted: boolean;
}
