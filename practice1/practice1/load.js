import http from 'k6/http';
import { check } from 'k6';


export const options = {
    summaryTrendStats: ['p(50)', 'p(95)', 'p(99)', 'max'],
  scenarios: {
    constant_rate: {
      executor: 'constant-arrival-rate',
      rate: 500,           // Задаем ровно 500 RPS
      timeUnit: '1s',
      duration: '30s',
      preAllocatedVUs: 100,
      maxVUs: 500,
    },
  },
};

export default function (){
    const res = http.get('http://localhost:8000/api/work?delayMs-20');
    check(res, { '200 OK ': (r) => r.status === 200 });
}