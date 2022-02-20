var question_ids = []
async function main(idToken){
    question_ids = []
    let request = await fetch("/manager/answers", {
        method: 'GET',
        headers: {
            'Accept': 'application/json',
            'Authorization': 'Token '+idToken
        }
    })
    let response = await request.json();
    var bars = {'Total':false, '':false}
    var answers = {}
    var totals = {"count":0}
    console.log(response)
    response.forEach(answer => {
        bars[answer.date] = false
        if(totals[answer.facet] == undefined) {
            totals[answer.facet] = 0
        }
        totals[answer.facet] += answer.total
        totals.count += answer.count
        if(answers[answer.facet] == undefined){
            answers[answer.facet] = []
        }
        answers[answer.facet].push(answer.total/answer.count)
    })
    // because count is added each facet
    totals.count /= (Object.keys(totals).length-1)
    for (let [key, value] of Object.entries(totals)) {
        if(key == 'count') {
            continue
        }
        answers[key].unshift(NaN)
        answers[key].unshift(value/totals.count)
    }
    var series = []
    for (let [facet, value] of Object.entries(answers)) {
        let dict = {"name": facet, "data":value}
        series.push(dict)
    }
    var categories = []
    for (let key in bars) {
        categories.push(key)
    }
    console.log(categories)
    console.log(JSON.stringify(series))

    Highcharts.chart('container', {
        chart: {
          type: 'column'
        },
        title: {
          text: 'Well Being of Team 1'
        },
        xAxis: {
            //dates
          categories: categories
        },
        yAxis: {
          min: 0,
          stackLabels: {
            enabled: true,
            style: {
              fontWeight: 'bold',
              color: ( // theme
                Highcharts.defaultOptions.title.style &&
                Highcharts.defaultOptions.title.style.color
              ) || 'gray'
            }
          }
        },
        legend: {
          align: 'right',
          x: -30,
          verticalAlign: 'top',
          y: 25,
          floating: true,
          backgroundColor:
            Highcharts.defaultOptions.legend.backgroundColor || 'white',
          borderColor: '#CCC',
          borderWidth: 1,
          shadow: false
        },
        tooltip: {
          headerFormat: '<b>{point.x}</b><br/>',
          pointFormat: '{series.name}: {point.y}<br/>Total: {point.stackTotal}'
        },
        plotOptions: {
          column: {
            stacking: 'normal',
            dataLabels: {
              enabled: true
            }
          }
        },
        series: series
      });
}

initialize( function(idToken){
    main(idToken)
})