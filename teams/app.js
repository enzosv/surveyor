var question_ids = []
async function manager(idToken){
    question_ids = []
    let request = await fetch("/manager/answers", {
        method: 'GET',
        headers: {
            'Accept': 'application/json',
            'Authorization': 'Token '+idToken
        }
    })
    let response = await request.json();
    var bars = {'Total':false, '':false} // add space
    var answers = {}
    var totals = {"count":0}
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
        // add space
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

    Highcharts.chart('container', {
        lang: {
          decimalPoint: ',',
          thousandsSep: '.'
        },
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
          pointFormat: '{series.name}: {point.y}<br/>Total: {point.stackTotal}',
          valueDecimals: 2 
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

async function members(idToken) {
    let request = await fetch("/user/memberships", {
        method: 'GET',
        headers: {
            'Accept': 'application/json',
            'Authorization': 'Token '+idToken
        }
    })
    let response = await request.json();
    console.log(response)
    let container = document.getElementById("memberships")
    container.innerHTML = "<p>Organizations:"
    response.forEach(org => {
        var teamsDiv = ""
        org.teams.forEach(team => {
            
            var membersDiv = ""
            team.members.forEach(member => {
                membersDiv += `&nbsp&nbsp&nbsp&nbsp${member.username} ${(member.is_manager) ? " (Manager)": ""}<br>`
            }) 
            teamsDiv += `
                &nbsp&nbsp&nbsp${team.team_name}<br>
                &nbsp&nbsp&nbspMembers:<br>
                ${membersDiv}
            `
        })
        container.innerHTML += `
            &nbsp${org.organization_name}<br>
            &nbsp&nbspTeams:<br>
            ${teamsDiv}
        `
    })
    container.innerHTML += "</p>"
    console.log(JSON.stringify(response))
}

initialize( function(idToken){
    manager(idToken)
    members(idToken)
})