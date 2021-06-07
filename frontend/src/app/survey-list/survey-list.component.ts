import {Component, OnInit} from '@angular/core';
import {DataService} from '../data.service';
import {ToastrService} from 'ngx-toastr';
import {Survey} from '../Survey';
import {QuestionsAndAnswers} from '../QuestionsAndAnswers';
import {ActivatedRoute} from '@angular/router';


@Component({
  selector: 'app-survey-list',
  templateUrl: './survey-list.component.html',
  styleUrls: ['./survey-list.component.css']
})
export class SurveyListComponent implements OnInit {

  surveyId = null;
  survey = new Survey();
  questions = new Array<QuestionsAndAnswers>();

  constructor(public dataService: DataService,
              public toastr: ToastrService,
              public route: ActivatedRoute) {
  }

  options = {
    animation: false
  };

  ngOnInit(): void {
    this.route.queryParams.subscribe(params => {
      this.surveyId = params['surveyId'];
    });

    this.loadData();

    // Set refresh timer
    setInterval(() => {
      this.loadData();
    }, 3000);
  }

  selected(question: QuestionsAndAnswers, answerId: number): void {
    this.dataService.vote(this.surveyId, question.QuestionId, answerId)
      .subscribe((data) => {
        question.Voted = true;
        this.toastr.success('Thanks for your vote', '');
        this.loadData();
      });
  }

  recreate(): void {
    this.dataService.delete().subscribe(() => {
      this.dataService.recreate().subscribe((d) => {
        this.surveyId = d;

        // Load data
        this.loadData();
      });
    });
  }

  loadData(): void {
    if (this.surveyId !== undefined) {
      this.dataService.getSurvey(this.surveyId).subscribe((data) => {
        const tmp = Survey.fromJSON(JSON.parse(JSON.stringify(data)));

        if (this.survey.Name === undefined) {
          this.survey.Name = tmp.Name;
        }

        // Map questions and answers
        tmp.QuestionsAndAnswers.forEach(q => {
          let target = this.survey[q.QuestionId];
          if (target == undefined) {
            target = q;
            this.survey[q.QuestionId] = target;
            this.questions.push(target);
          }

          // Create chart data set
          target.VoteLabels = [];
          target.Votes = [];
          q.Answers.forEach(item => {
            target.VoteLabels.push(item.AnswerId + 1);
            target.Votes.push(item.Vote);
          });
        });
      }, error => {
        console.log('Error while running loaddata:' + error);
        this.surveyId = undefined;
        this.survey = new Survey();
        this.questions = new Array<QuestionsAndAnswers>();
      });
    }
  }
}
