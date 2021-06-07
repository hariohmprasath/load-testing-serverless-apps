import {Injectable} from '@angular/core';
import {HttpClient} from '@angular/common/http';
import {environment} from '../environments/environment';

@Injectable({
  providedIn: 'root'
})
export class DataService {
  survey = {};
  baseUrl = environment.baseUrl;

  constructor(private http: HttpClient) {
  }

  public delete(){
    return this.http.delete(this.baseUrl);
  }

  public recreate() {
    return this.http.put(this.baseUrl + '?recreate=true', '', {responseType: 'text'});
  }

  public vote(surveyId: string, questionId: number, answerId: number) {
    return this.http.put(this.baseUrl + '?surveyId=' + surveyId + '&questionId=' + questionId + '&answerId=' + answerId, '');
  }

  public getSurvey(surveyId: string) {
    return this.http.get(this.baseUrl + '?surveyId=' + surveyId);
  }
}
