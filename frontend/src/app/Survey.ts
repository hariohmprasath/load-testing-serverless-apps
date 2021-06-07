import {QuestionsAndAnswers} from './QuestionsAndAnswers';

export class Survey {
  public Name: string;
  public QuestionsAndAnswers: QuestionsAndAnswers[];

  // tslint:disable-next-line:ban-types
  static fromJSON(d: Object): Survey {
    return Object.assign(new Survey(), d);
  }
}
