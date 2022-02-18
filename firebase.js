
const firebaseConfig = {
    apiKey: "AIzaSyB3QQJTRaXpWRO-HRcF5kvWZ5WDKqoHSQY",
    authDomain: "nikki-db42b.firebaseapp.com",
    projectId: "nikki-db42b",
    storageBucket: "nikki-db42b.appspot.com",
    messagingSenderId: "998456593406",
    appId: "1:998456593406:web:ad1475c13c53e8fb4eceaf",
    measurementId: "G-W4RYWR8FB1"
};
const app = firebase.initializeApp(firebaseConfig);
var token

async function authenticate(success, failure) {
    firebase.auth().onAuthStateChanged((user) => {
        if(!user){
            if(window.location.href != window.location.origin + "/") {
                console.log("redirect")
                window.location.replace("/");
                return
            }
            signin()
            return
        }
        user.getIdToken(true).then(function(idToken) {
            success(idToken)
        }).catch(function(error) {
            signin()
            return
            failure(error)
        });
    });
}

function signout() {
    firebase.auth().signOut().then(() => {
        window.location.replace("/");
      }).catch((error) => {
          console.error(error)
      });
}


authenticate(function(idToken){
    token = idToken
    if(window.location.href == window.location.origin + "/") {
        window.location.replace("/questions");
        return
    }
}, function(error){
    console.error(error)
    // console.log(window.location.href, window.location.origin)
    // if(window.location.href != window.location.origin + "/") {
    //     console.log("redirect")
    //     window.location.replace("/");
    //     return
    // }
    // signin()
})


function signin() {
    var ui = new firebaseui.auth.AuthUI(firebase.auth());
    var uiConfig = {
        callbacks: {
            signInSuccessWithAuthResult: function (authResult, redirectUrl) {
                // User successfully signed in.
                // Return type determines whether we continue the redirect automatically
                // or whether we leave that to developer to handle.
                console.log(authResult.credential.idToken)
                firebase.auth().currentUser.getIdToken(/* forceRefresh */ true).then(function (idToken) {
                    fetch("/signin", {
                        method: 'POST',
                        headers: {
                            'Accept': 'application/json',
                            'Authorization': 'Token ' + idToken
                        }
                    })
                }).catch(function (error) {
                    // Handle error
                });
                return true;
            }
        },
        // Will use popup for IDP Providers sign-in flow instead of the default, redirect.
        signInFlow: 'popup',
        signInSuccessUrl: '/questions',
        signInOptions: [
            {
                provider: firebase.auth.GoogleAuthProvider.PROVIDER_ID,
                requireDisplayName: true
            },
            {
                provider: firebase.auth.EmailAuthProvider.PROVIDER_ID,
                requireDisplayName: true
            }
        ],
        // Terms of service url.
        tosUrl: '<your-tos-url>',
        // Privacy policy url.
        privacyPolicyUrl: '<your-privacy-policy-url>'
    };
    firebase.auth().setPersistence(firebase.auth.Auth.Persistence.LOCAL)
    .then(() => {
        ui.start('#firebaseui-auth-container', uiConfig);
    })
    .catch((error)=>{
        console.error(error)
    })
}